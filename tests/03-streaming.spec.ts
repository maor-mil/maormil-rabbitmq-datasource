import { test, expect } from './fixtures';
import { TIMESTAMP_FIELD, uniqueId } from './helpers/grafana';

const STREAM_TIMEOUT = { timeout: 30_000 };

test.describe('streaming', () => {
  test('JSON messages published to the stream appear in a table panel', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    const marker = uniqueId('msg');
    await rabbitmq.publishToStream(env.streamName, { name: marker, value: 42, ok: true });

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
    await expect(panel.fieldNames).toContainText([TIMESTAMP_FIELD, 'name', 'value', 'ok']);
  });

  test('messages keep arriving after the first one', async ({ createStreamingEnv, gotoDashboardPage, rabbitmq }) => {
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    for (let i = 0; i < 3; i++) {
      const marker = uniqueId(`msg${i}`);
      await rabbitmq.publishToStream(env.streamName, { name: marker, value: i });
      await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
    }
  });

  test('messages published to a configured exchange reach the stream through the configured binding', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const env = await createStreamingEnv({
      topology: ({ id, streamName }) => ({
        exchangesOptions: [
          {
            name: `${id}.exchange`,
            type: 'fanout',
            durable: true,
            autoDeleted: false,
            internal: false,
            noWait: false,
            shouldDisposeExchange: true,
            disposeIfUnused: true,
          },
        ],
        bindingsOptions: [
          {
            senderName: `${id}.exchange`,
            routingKey: '/',
            receiverName: streamName,
            isQueueBinding: true,
            noWait: false,
            shouldDisposeBinding: true,
          },
        ],
      }),
    });
    const exchange = `${env.id}.exchange`;
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);
    await expect
      .poll(async () => (await rabbitmq.bindingsFrom(exchange)).some((b) => b.destination === env.streamName))
      .toBe(true);

    const marker = uniqueId('via-exchange');
    expect(await rabbitmq.publish({ exchange, routingKey: 'any', payload: { name: marker } })).toBe(true);

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
  });

  test('a non-JSON message does not stop the stream', async ({ createStreamingEnv, gotoDashboardPage, rabbitmq }) => {
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    await rabbitmq.publishToStream(env.streamName, 'this is not json');
    const marker = uniqueId('after-garbage');
    await rabbitmq.publishToStream(env.streamName, { name: marker });

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
    await expect(panel.locator.getByText('this is not json')).toHaveCount(0);
  });

  test('nested objects and arrays are kept as JSON values', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    const marker = uniqueId('nested');
    await rabbitmq.publishToStream(env.streamName, { name: marker, nested: { inner: 'deep-value' }, list: [7, 8] });

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
    await expect(panel.fieldNames).toContainText(['name', 'nested', 'list']);
    await expect(panel.locator.getByText('deep-value')).toBeVisible();
  });
});
