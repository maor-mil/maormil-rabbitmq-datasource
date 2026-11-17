import { test, expect } from './fixtures';
import { expectKnownBug, KNOWN_BUGS } from './known-bugs';
import { uniqueId } from './helpers/grafana';

const STREAM_TIMEOUT = { timeout: 30_000 };

// These cases exercise the JSON → data frame conversion (pkg/plugin/framer.go) with messages whose shape
// changes over time. Each test uses its own data source and stream, so a message that breaks the plugin
// cannot be replayed into other tests. Keep this file last: a crashing plugin process restarts every
// data source instance.
test.describe('message edge cases', () => {
  test('a field whose type changes between messages keeps receiving values', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    expectKnownBug(KNOWN_BUGS.typeChangeDropsValues);
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    await rabbitmq.publishToStream(env.streamName, { name: uniqueId('number'), reading: 1 });
    const marker = uniqueId('now-a-string');
    await rabbitmq.publishToStream(env.streamName, { name: 'string-reading', reading: marker });

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
  });

  test('a null value after a non-null value does not break the stream', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    expectKnownBug(KNOWN_BUGS.nullCrashesPlugin);
    const env = await createStreamingEnv();
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    await rabbitmq.publishToStream(env.streamName, { name: uniqueId('with-value'), reading: 1 });
    await rabbitmq.publishToStream(env.streamName, { name: uniqueId('with-null'), reading: null });
    const marker = uniqueId('after-null');
    await rabbitmq.publishToStream(env.streamName, { name: marker, reading: 2 });

    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
  });
});
