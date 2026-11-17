import { test, expect } from './fixtures';
import { deleteDatasource, uniqueId } from './helpers/grafana';

const STREAM_TIMEOUT = { timeout: 30_000 };

test.describe('resilience and lifecycle', () => {
  test.describe.configure({ timeout: 120_000 });

  test('the stream is recreated and streaming resumes after the stream is deleted in RabbitMQ', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
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

  test('streaming resumes after RabbitMQ drops the connections, without losing stored messages', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    const before = uniqueId('before-outage');
    await rabbitmq.publishToStream(env.streamName, { name: before });
    await expect(panel.locator.getByText(before)).toBeVisible(STREAM_TIMEOUT);

    expect(await rabbitmq.closeAllConnections()).toBeGreaterThan(0);

    // README: "If the RabbitMQ is down the plugin will try to keep reconnecting to the RabbitMQ".
    // Keep publishing until a message makes it to the panel again (the stream may be briefly unavailable).
    await expect
      .poll(
        async () => {
          const after = uniqueId('after-outage');
          if (await rabbitmq.queueExists(env.streamName)) {
            await rabbitmq.publish({ routingKey: env.streamName, payload: { name: after } });
          }
          return panel.locator.getByText(after).isVisible({ timeout: 5_000 }).catch(() => false);
        },
        { timeout: 90_000, intervals: [5_000] }
      )
      .toBe(true);
    // A connection loss must not delete the stream: the message published before the outage is still stored.
    await expect.poll(() => rabbitmq.messageCount(env.streamName)).toBeGreaterThanOrEqual(2);
  });

  test('closing the dashboard does not delete the stream', async ({
    createStreamingEnv,
    gotoDashboardPage,
    page,
    rabbitmq,
  }) => {
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

  test('the plugin never deletes RabbitMQ objects, not even when the data source is deleted', async ({
    createStreamingEnv,
    gotoDashboardPage,
    page,
    rabbitmq,
  }) => {
    const env = await createStreamingEnv({
      topology: ({ id, streamName }) => ({
        exchangesOptions: [
          { name: `${id}.source`, type: 'fanout', durable: true, autoDeleted: false, internal: false, noWait: false },
          { name: `${id}.target`, type: 'fanout', durable: true, autoDeleted: false, internal: false, noWait: false },
        ],
        bindingsOptions: [
          { senderName: `${id}.target`, routingKey: '/', receiverName: streamName, isQueueBinding: true, noWait: false },
          { senderName: `${id}.source`, routingKey: '/', receiverName: `${id}.target`, isQueueBinding: false, noWait: false },
        ],
      }),
    });
    const source = `${env.id}.source`;
    const target = `${env.id}.target`;

    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);
    // Bindings are declared right after the stream.
    await expect.poll(async () => (await rabbitmq.bindingsFrom(source)).length).toBe(1);
    const marker = uniqueId('kept');
    expect(await rabbitmq.publish({ exchange: source, routingKey: 'x', payload: { name: marker } })).toBe(true);
    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);

    // Delete the data source while its panel is streaming, then leave the dashboard so the stream stops.
    await deleteDatasource(page.request, env.dsUid);
    await page.goto('/');
    await page.waitForTimeout(30_000);

    expect(await rabbitmq.queueExists(env.streamName)).toBe(true);
    expect(await rabbitmq.messageCount(env.streamName)).toBeGreaterThanOrEqual(1);
    expect(await rabbitmq.exchangeExists(source)).toBe(true);
    expect(await rabbitmq.exchangeExists(target)).toBe(true);
    expect((await rabbitmq.bindingsFrom(source)).map((b) => b.destination)).toEqual([target]);
    expect((await rabbitmq.bindingsFrom(target)).map((b) => b.destination)).toEqual([env.streamName]);
  });
});
