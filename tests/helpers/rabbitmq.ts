import { APIRequestContext, expect, request } from '@playwright/test';

/** Management HTTP API of the broker started by docker-compose.yaml, as seen from the test runner. */
export const RABBITMQ_MANAGEMENT_URL = process.env.RABBITMQ_MANAGEMENT_URL ?? 'http://localhost:15672';
const RABBITMQ_MANAGEMENT_USER = process.env.RABBITMQ_MANAGEMENT_USER ?? 'guest';
const RABBITMQ_MANAGEMENT_PASSWORD = process.env.RABBITMQ_MANAGEMENT_PASSWORD ?? 'guest';

const DEFAULT_VHOST = '/';

export type PublishOptions = {
  /** Exchange to publish to. Defaults to the default exchange, which routes by queue/stream name. */
  exchange?: string;
  routingKey: string;
  /** Objects are JSON encoded; strings are sent as-is. */
  payload: unknown;
  /** AMQP 0.9.1 headers. Non `x-` headers become AMQP 1.0 application properties in streams. */
  headers?: Record<string, string | number | boolean>;
  vhost?: string;
};

type Binding = { source: string; destination: string; destination_type: string; routing_key: string };

/**
 * Thin client for the RabbitMQ management HTTP API, used to publish test messages and to
 * inspect/clean up the topology (streams, exchanges, bindings) the plugin declares.
 */
export class RabbitMQManagement {
  private constructor(private readonly api: APIRequestContext) {}

  static async create(): Promise<RabbitMQManagement> {
    const api = await request.newContext({
      baseURL: RABBITMQ_MANAGEMENT_URL,
      httpCredentials: { username: RABBITMQ_MANAGEMENT_USER, password: RABBITMQ_MANAGEMENT_PASSWORD, send: 'always' },
    });
    return new RabbitMQManagement(api);
  }

  async dispose(): Promise<void> {
    await this.api.dispose();
  }

  /** Publishes one message and returns whether the broker routed it to at least one queue/stream. */
  async publish({ exchange = '', routingKey, payload, headers, vhost = DEFAULT_VHOST }: PublishOptions): Promise<boolean> {
    const exchangeName = exchange === '' ? 'amq.default' : exchange;
    const res = await this.api.post(`/api/exchanges/${enc(vhost)}/${enc(exchangeName)}/publish`, {
      data: {
        properties: headers ? { headers } : {},
        routing_key: routingKey,
        payload: typeof payload === 'string' ? payload : JSON.stringify(payload),
        payload_encoding: 'string',
      },
    });
    expect(res.ok(), `publish to "${exchangeName}" failed: ${await res.text()}`).toBeTruthy();
    return (await res.json()).routed === true;
  }

  /** Publishes directly into a stream (via the default exchange) and fails if the stream does not exist. */
  async publishToStream(stream: string, payload: unknown, headers?: PublishOptions['headers']): Promise<void> {
    const routed = await this.publish({ routingKey: stream, payload, headers });
    expect(routed, `message was not routed: stream "${stream}" does not exist`).toBeTruthy();
  }

  async queueExists(name: string, vhost = DEFAULT_VHOST): Promise<boolean> {
    const res = await this.api.get(`/api/queues/${enc(vhost)}/${enc(name)}`);
    return res.status() === 200;
  }

  /** Returns the queue type ("stream", "classic", "quorum") or undefined when the queue does not exist. */
  async queueType(name: string, vhost = DEFAULT_VHOST): Promise<string | undefined> {
    const res = await this.api.get(`/api/queues/${enc(vhost)}/${enc(name)}`);
    return res.status() === 200 ? (await res.json()).type : undefined;
  }

  /** Waits until the plugin has declared the given stream/queue. */
  async waitForQueue(name: string, timeout = 30_000): Promise<void> {
    await expect.poll(() => this.queueExists(name), { timeout, message: `queue "${name}" was never declared` }).toBe(true);
  }

  async deleteQueue(name: string, vhost = DEFAULT_VHOST): Promise<void> {
    const res = await this.api.delete(`/api/queues/${enc(vhost)}/${enc(name)}`);
    expect([204, 404], `deleting queue "${name}" failed: ${await res.text()}`).toContain(res.status());
  }

  /** Approximate number of messages stored in a queue/stream (0 when it does not exist). */
  async messageCount(name: string, vhost = DEFAULT_VHOST): Promise<number> {
    const res = await this.api.get(`/api/queues/${enc(vhost)}/${enc(name)}`);
    return res.status() === 200 ? ((await res.json()).messages ?? 0) : 0;
  }

  /** Force-closes every client connection (AMQP and stream protocol), simulating a broker outage. Returns the count. */
  async closeAllConnections(reason = 'e2e: simulated outage'): Promise<number> {
    // The management API lists connections with its stats interval (~5s), so wait until they show up.
    let connections: Array<{ name: string }> = [];
    await expect
      .poll(
        async () => {
          const list = await this.api.get('/api/connections');
          connections = list.ok() ? await list.json() : [];
          return connections.length;
        },
        { timeout: 20_000, message: 'no client connections listed by RabbitMQ' }
      )
      .toBeGreaterThan(0);
    for (const { name } of connections) {
      await this.api.delete(`/api/connections/${enc(name)}`, { headers: { 'X-Reason': reason } });
    }
    return connections.length;
  }

  async declareExchange(name: string, type: 'direct' | 'fanout' | 'topic' | 'headers', vhost = DEFAULT_VHOST): Promise<void> {
    const res = await this.api.put(`/api/exchanges/${enc(vhost)}/${enc(name)}`, { data: { type, durable: false } });
    expect(res.ok(), await res.text()).toBeTruthy();
  }

  async declareQueue(name: string, type: 'classic' | 'quorum' | 'stream', vhost = DEFAULT_VHOST): Promise<void> {
    const res = await this.api.put(`/api/queues/${enc(vhost)}/${enc(name)}`, {
      data: { durable: type !== 'classic', arguments: { 'x-queue-type': type } },
    });
    expect(res.ok(), await res.text()).toBeTruthy();
  }

  async bindQueue(exchange: string, queue: string, routingKey: string, vhost = DEFAULT_VHOST): Promise<void> {
    const res = await this.api.post(`/api/bindings/${enc(vhost)}/e/${enc(exchange)}/q/${enc(queue)}`, {
      data: { routing_key: routingKey },
    });
    expect(res.ok(), await res.text()).toBeTruthy();
  }

  async exchangeExists(name: string, vhost = DEFAULT_VHOST): Promise<boolean> {
    const res = await this.api.get(`/api/exchanges/${enc(vhost)}/${enc(name)}`);
    return res.status() === 200;
  }

  async deleteExchange(name: string, vhost = DEFAULT_VHOST): Promise<void> {
    const res = await this.api.delete(`/api/exchanges/${enc(vhost)}/${enc(name)}`);
    expect([204, 404], `deleting exchange "${name}" failed: ${await res.text()}`).toContain(res.status());
  }

  /** Bindings whose source is the given exchange. */
  async bindingsFrom(exchange: string, vhost = DEFAULT_VHOST): Promise<Binding[]> {
    const res = await this.api.get(`/api/exchanges/${enc(vhost)}/${enc(exchange)}/bindings/source`);
    if (res.status() === 404) {
      return [];
    }
    expect(res.ok(), await res.text()).toBeTruthy();
    return res.json();
  }

  async createUser(username: string, password: string, vhost = DEFAULT_VHOST): Promise<void> {
    const user = await this.api.put(`/api/users/${enc(username)}`, { data: { password, tags: '' } });
    expect(user.ok(), await user.text()).toBeTruthy();
    const perms = await this.api.put(`/api/permissions/${enc(vhost)}/${enc(username)}`, {
      data: { configure: '.*', write: '.*', read: '.*' },
    });
    expect(perms.ok(), await perms.text()).toBeTruthy();
  }

  async deleteUser(username: string): Promise<void> {
    const res = await this.api.delete(`/api/users/${enc(username)}`);
    expect([204, 404]).toContain(res.status());
  }
}

function enc(value: string): string {
  return encodeURIComponent(value);
}
