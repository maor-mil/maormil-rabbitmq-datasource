import { test, expect } from './fixtures';
import { uniqueId } from './helpers/grafana';

const STREAM_TIMEOUT = { timeout: 30_000 };

// https://www.rabbitmq.com/docs/stream-filtering
test.describe('stream filtering', () => {
  test.describe.configure({ timeout: 120_000 });

  test('Bloom filter values and SQL filter expressions select messages; invalid SQL is reported', async ({
    createStreamingEnv,
    gotoDashboardPage,
    rabbitmq,
  }) => {
    const stream = `${uniqueId()}.regions`;
    await rabbitmq.declareQueue(stream, 'stream');

    try {
      // AMQP 0.9.1 publishes: x-stream-filter-value feeds the Bloom filter, "region" becomes an application
      // property that SQL filters can read.
      const markers: Record<string, string> = {};
      for (const region of ['emea', 'apac', 'amer']) {
        markers[region] = uniqueId(region);
        await rabbitmq.publishToStream(
          stream,
          { name: markers[region] },
          { 'x-stream-filter-value': region, region: region.toUpperCase() }
        );
      }

      const env = await createStreamingEnv({
        panels: [
          { title: 'Bloom', query: { source: 'stream', stream, offset: 'first', filterValues: ['emea'] } },
          { title: 'SQL', query: { source: 'stream', stream, offset: 'first', sql: "region IN ('APAC', 'AMER')" } },
          { title: 'Broken SQL', query: { source: 'stream', stream, offset: 'first', sql: "region ==== 'APAC'" } },
        ],
      });
      const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
      const bloom = dashboard.getPanelByTitle('Bloom');
      const sql = dashboard.getPanelByTitle('SQL');

      await expect(bloom.locator.getByText(markers.emea)).toBeVisible(STREAM_TIMEOUT);
      await expect(sql.locator.getByText(markers.apac)).toBeVisible(STREAM_TIMEOUT);
      await expect(sql.locator.getByText(markers.amer)).toBeVisible(STREAM_TIMEOUT);
      await expect(bloom.locator.getByText(markers.apac)).toHaveCount(0);
      await expect(bloom.locator.getByText(markers.amer)).toHaveCount(0);
      await expect(sql.locator.getByText(markers.emea)).toHaveCount(0);

      // Grafana 13 renders panels lazily: bring the third panel into view first.
      const broken = dashboard.getPanelByTitle('Broken SQL');
      await broken.scrollIntoView();
      await expect(broken.getErrorIcon()).toBeVisible(STREAM_TIMEOUT);
    } finally {
      await rabbitmq.deleteQueue(stream);
    }
  });
});
