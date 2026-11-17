import { Locator } from '@playwright/test';
import { test, expect } from './fixtures';
import { uniqueId } from './helpers/grafana';
import { RabbitMQManagement } from './helpers/rabbitmq';

const STREAM_TIMEOUT = { timeout: 30_000 };

const shows = (panel: { locator: Locator }, text: string, timeout = 3_000) =>
  panel.locator
    .getByText(text)
    .waitFor({ timeout })
    .then(
      () => true,
      () => false
    );

/** Queue consumers attach asynchronously (tap queues are declared when the panel starts streaming): keep
 * publishing fresh markers until one shows up in the panel. */
async function publishUntilShown(
  rabbitmq: RabbitMQManagement,
  panel: { locator: Locator },
  publish: (marker: string) => Promise<unknown>
) {
  await expect
    .poll(
      async () => {
        const marker = uniqueId('msg');
        await publish(marker);
        return shows(panel, marker);
      },
      { timeout: 60_000, intervals: [2_000] }
    )
    .toBe(true);
}

test.describe('per-panel sources', () => {
  test.describe.configure({ timeout: 120_000 });

  test('the query editor offers the sources and warns before consuming a queue', async ({
    panelEditPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);
    const editor = panelEditPage.getQueryEditorRow('A');

    await expect(editor.getByRole('radio', { name: 'Data source stream' })).toBeChecked();
    await editor.getByRole('radio', { name: 'Queue' }).check({ force: true });
    await expect(editor.getByRole('radio', { name: 'Tap an exchange' })).toBeChecked();
    await expect(editor.getByRole('textbox', { name: 'Exchange' })).toBeVisible();
    await expect(editor.getByText('Consuming a queue removes its messages')).toHaveCount(0);

    // The option descriptions are tooltips; move away so the "Queue" tooltip no longer covers the row below.
    await page.mouse.move(0, 0);
    await editor.getByRole('radio', { name: 'Consume a queue' }).check({ force: true });
    await expect(editor.getByText('Consuming a queue removes its messages')).toBeVisible();
  });

  test('tapping an exchange shows its messages without taking them from application queues', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const exchange = `${uniqueId()}.topic`;
    const appQueue = `${uniqueId()}.app`;
    await rabbitmq.declareExchange(exchange, 'topic');
    await rabbitmq.declareQueue(appQueue, 'classic');
    await rabbitmq.bindQueue(exchange, appQueue, 'sensors.#');

    try {
      const env = await createStreamingEnv({
        panels: [{ title: 'Tap', query: { source: 'queue', queueMode: 'tap', exchange, routingKey: 'sensors.#' } }],
      });
      const panel = (await gotoDashboardPage({ uid: env.dashboardUid })).getPanelByTitle('Tap');

      await publishUntilShown(rabbitmq, panel, (name) =>
        rabbitmq.publish({ exchange, routingKey: 'sensors.temperature', payload: { name } })
      );

      // The application queue still received every message.
      await expect.poll(() => rabbitmq.messageCount(appQueue), STREAM_TIMEOUT).toBeGreaterThan(0);
    } finally {
      await rabbitmq.deleteQueue(appQueue);
      await rabbitmq.deleteExchange(exchange);
    }
  });

  test('consuming an existing quorum queue shows and removes its messages', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const queue = `${uniqueId()}.quorum`;
    await rabbitmq.declareQueue(queue, 'quorum');

    try {
      const marker = uniqueId('queued');
      await rabbitmq.publish({ routingKey: queue, payload: { name: marker } });
      const env = await createStreamingEnv({
        panels: [{ title: 'Queue', query: { source: 'queue', queueMode: 'existing', queue } }],
      });
      const panel = (await gotoDashboardPage({ uid: env.dashboardUid })).getPanelByTitle('Queue');

      await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
      await expect.poll(() => rabbitmq.messageCount(queue), STREAM_TIMEOUT).toBe(0);
    } finally {
      await rabbitmq.deleteQueue(queue);
    }
  });

  test('two panels stream two different sources independently', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const other = `${uniqueId()}.other`;
    await rabbitmq.declareQueue(other, 'stream');

    try {
      const env = await createStreamingEnv({
        panels: [
          { title: 'Own stream', query: {} },
          { title: 'Other stream', query: { source: 'stream', stream: other, offset: 'first' } },
        ],
      });
      const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
      const own = dashboard.getPanelByTitle('Own stream');
      const otherPanel = dashboard.getPanelByTitle('Other stream');
      await rabbitmq.waitForQueue(env.streamName);

      const ownMarker = uniqueId('own');
      const otherMarker = uniqueId('other');
      await rabbitmq.publishToStream(env.streamName, { name: ownMarker });
      await rabbitmq.publishToStream(other, { name: otherMarker });

      await expect(own.locator.getByText(ownMarker)).toBeVisible(STREAM_TIMEOUT);
      await expect(otherPanel.locator.getByText(otherMarker)).toBeVisible(STREAM_TIMEOUT);
      await expect(own.locator.getByText(otherMarker)).toHaveCount(0);
      await expect(otherPanel.locator.getByText(ownMarker)).toHaveCount(0);
    } finally {
      await rabbitmq.deleteQueue(other);
    }
  });

  test('an incomplete query shows an error on the panel', async ({ createStreamingEnv, gotoDashboardPage }) => {
    const env = await createStreamingEnv({ panels: [{ title: 'Broken', query: { source: 'stream' } }] });
    const panel = (await gotoDashboardPage({ uid: env.dashboardUid })).getPanelByTitle('Broken');
    await expect(panel.getErrorIcon()).toBeVisible(STREAM_TIMEOUT);
  });
});
