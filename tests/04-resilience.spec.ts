import { test, expect } from './fixtures';
import { expectKnownBug, KNOWN_BUGS } from './known-bugs';
import { deleteDatasource, uniqueId } from './helpers/grafana';

const STREAM_TIMEOUT = { timeout: 30_000 };

test.describe('resilience and lifecycle', () => {
  test.describe.configure({ timeout: 120_000 });

  test('the stream is recreated and streaming resumes after the stream is deleted in RabbitMQ', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    expectKnownBug(KNOWN_BUGS.streamNotRecreated);
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    const before = uniqueId('before-delete');
    await rabbitmq.publishToStream(env.streamName, { name: before });
    await expect(panel.locator.getByText(before)).toBeVisible(STREAM_TIMEOUT);

    await rabbitmq.deleteQueue(env.streamName);
    // README: "If the stream is deleted in the RabbitMQ itself - the stream, its consumer, and the
    // pre-configured exchanges and bindings will be recreated."
    await rabbitmq.waitForQueue(env.streamName, 60_000);

    const after = uniqueId('after-delete');
    await rabbitmq.publishToStream(env.streamName, { name: after });
    await expect(panel.locator.getByText(after)).toBeVisible(STREAM_TIMEOUT);
  });

  test('closing the dashboard does not delete the stream', async ({
    createStreamingEnv,
    gotoDashboardPage,
    page,
    rabbitmq,
  }) => {
    expectKnownBug(KNOWN_BUGS.dashboardCloseDeletesStream);
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);
    const marker = uniqueId('kept');
    await rabbitmq.publishToStream(env.streamName, { name: marker });
    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);

    // Leave the dashboard. Grafana Live stops the stream (cancels RunStream) once it has no subscribers.
    await page.goto('/');
    await page.waitForTimeout(30_000);

    // The data source still exists, so its stream (and the messages in it) must still be there.
    expect(await rabbitmq.queueExists(env.streamName)).toBe(true);
  });

  test('deleting the data source disposes its stream and bindings when disposal is enabled', async ({
    createStreamingEnv,
    gotoDashboardPage,
    page,
    rabbitmq,
  }) => {
    expectKnownBug(KNOWN_BUGS.exchangeBindingNotRemoved);
    const env = await createStreamingEnv({
      topology: ({ id, streamName }) => ({
        exchangesOptions: [
          {
            name: `${id}.source`,
            type: 'fanout',
            durable: true,
            autoDeleted: false,
            internal: false,
            noWait: false,
            shouldDisposeExchange: false,
            disposeIfUnused: false,
          },
          {
            name: `${id}.target`,
            type: 'fanout',
            durable: true,
            autoDeleted: false,
            internal: false,
            noWait: false,
            shouldDisposeExchange: false,
            disposeIfUnused: false,
          },
        ],
        bindingsOptions: [
          {
            senderName: `${id}.target`,
            routingKey: '/',
            receiverName: streamName,
            isQueueBinding: true,
            noWait: false,
            shouldDisposeBinding: true,
          },
          {
            senderName: `${id}.source`,
            routingKey: '/',
            receiverName: `${id}.target`,
            isQueueBinding: false,
            noWait: false,
            shouldDisposeBinding: true,
          },
        ],
      }),
    });
    const source = `${env.id}.source`;
    const target = `${env.id}.target`;

    // README: the consumer must have been created (a panel opened) for disposal to happen.
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);
    const marker = uniqueId('dispose');
    expect(await rabbitmq.publish({ exchange: source, routingKey: 'x', payload: { name: marker } })).toBe(true);
    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);

    // Delete the data source while its panel is still streaming, then leave the dashboard.
    await deleteDatasource(page.request, env.dsUid);
    await page.goto('/');

    await expect.poll(() => rabbitmq.queueExists(env.streamName), { timeout: 60_000 }).toBe(false);
    // Both exchanges have shouldDisposeExchange=false, so they stay; the bindings (shouldDisposeBinding=true)
    // must be removed, including the exchange-to-exchange one.
    expect(await rabbitmq.exchangeExists(source)).toBe(true);
    expect(await rabbitmq.exchangeExists(target)).toBe(true);
    await expect.poll(async () => (await rabbitmq.bindingsFrom(source)).length, { timeout: 30_000 }).toBe(0);
  });
});
