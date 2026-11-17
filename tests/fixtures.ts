import { test as base, expect } from '@grafana/plugin-e2e';
import { BindingOptions, ExchangeOptions, RabbitMQDataSourceOptions, RabbitMQSecureJsonData, StreamOptions } from '../src/types';
import {
  createStreamingDashboard,
  datasourceJsonData,
  deleteDashboard,
  deleteDatasource,
  DS_TYPE,
  STREAM_PANEL_TITLE,
  uniqueId,
} from './helpers/grafana';
import { RabbitMQManagement } from './helpers/rabbitmq';

export type StreamingEnv = {
  id: string;
  dsUid: string;
  dashboardUid: string;
  streamName: string;
  panelTitle: string;
};

export type StreamingEnvOptions = {
  streamOptions?: Partial<StreamOptions>;
  /** Connection settings (ports, TLS, authentication) on top of the defaults. */
  connection?: Partial<RabbitMQDataSourceOptions>;
  /** Secrets; defaults to the guest password. */
  secureJsonData?: RabbitMQSecureJsonData;
  /** Built from the env id; receives the stream name so bindings can target it. */
  topology?: (env: { id: string; streamName: string }) => {
    exchangesOptions?: ExchangeOptions[];
    bindingsOptions?: BindingOptions[];
  };
};

type Fixtures = {
  /** Management API client for the compose RabbitMQ broker. */
  rabbitmq: RabbitMQManagement;
  /**
   * Creates an isolated data source (with its own stream) and a dashboard with a streaming Table panel.
   * Everything it creates in Grafana and RabbitMQ is removed after the test.
   */
  createStreamingEnv: (options?: StreamingEnvOptions) => Promise<StreamingEnv>;
};

export const test = base.extend<Fixtures>({
  rabbitmq: async ({}, use) => {
    const client = await RabbitMQManagement.create();
    await use(client);
    await client.dispose();
  },

  createStreamingEnv: async ({ createDataSource, page, rabbitmq }, use) => {
    const created: Array<{ env: StreamingEnv; exchanges: string[] }> = [];

    await use(async (options = {}) => {
      const id = uniqueId();
      const streamName = `${id}.stream`;
      const topology = options.topology?.({ id, streamName }) ?? {};
      const ds = await createDataSource({
        type: DS_TYPE,
        name: `RabbitMQ ${id}`,
        jsonData: datasourceJsonData({
          streamOptions: { ...options.streamOptions, streamName },
          connection: options.connection,
          ...topology,
        }),
        secureJsonData: options.secureJsonData ?? { password: 'guest' },
      });
      const dashboardUid = await createStreamingDashboard(page.request, { uid: id, title: `E2E ${id}`, dsUid: ds.uid });
      const env = { id, dsUid: ds.uid, dashboardUid, streamName, panelTitle: STREAM_PANEL_TITLE };
      created.push({ env, exchanges: (topology.exchangesOptions ?? []).map((e) => e.name) });
      return env;
    });

    for (const { env, exchanges } of created) {
      await deleteDashboard(page.request, env.dashboardUid);
      await deleteDatasource(page.request, env.dsUid);
      await rabbitmq.deleteQueue(env.streamName);
      for (const exchange of exchanges) {
        await rabbitmq.deleteExchange(exchange);
      }
    }
  },
});

export { expect };
