import { APIRequestContext, expect } from '@playwright/test';
import { BindingOptions, ExchangeOptions, RabbitMQDataSourceOptions, StreamOptions } from '../../src/types';
import pluginJson from '../../src/plugin.json';
import { DEFAULT_STREAM_OPTIONS, withDefaults } from '../../src/components/defaults';

export const DS_TYPE = pluginJson.id;

/** Name of the timestamp field the backend adds to every message. */
export const TIMESTAMP_FIELD = 'RmqMsgConsumedTimestamp';

/** Host of the RabbitMQ broker as seen from inside the Grafana container (docker compose service name). */
export const RABBITMQ_HOST_FROM_GRAFANA = process.env.RABBITMQ_HOST_FROM_GRAFANA ?? 'rabbitmq';

export const STREAM_PANEL_TITLE = 'Stream';

/** The plugin defaults (same as the config editor and provisioning), pointed at the compose broker. */
export function datasourceJsonData(overrides: {
  streamOptions: Partial<StreamOptions> & { streamName: string };
  exchangesOptions?: ExchangeOptions[];
  bindingsOptions?: BindingOptions[];
  connection?: Partial<RabbitMQDataSourceOptions>;
}): RabbitMQDataSourceOptions {
  return withDefaults({
    host: RABBITMQ_HOST_FROM_GRAFANA,
    ...overrides.connection,
    streamOptions: { ...DEFAULT_STREAM_OPTIONS, ...overrides.streamOptions },
    exchangesOptions: overrides.exchangesOptions ?? [],
    bindingsOptions: overrides.bindingsOptions ?? [],
  });
}

/**
 * Creates (or overwrites) a dashboard with a single Table panel that streams from the given data source.
 * Rows are sorted by the consumed timestamp (newest first) so new messages are always rendered at the top,
 * even when the stream replays older messages from its first offset.
 */
export async function createStreamingDashboard(
  api: APIRequestContext,
  { uid, title, dsUid }: { uid: string; title: string; dsUid: string }
): Promise<string> {
  const datasource = { type: DS_TYPE, uid: dsUid };
  const res = await api.post('/api/dashboards/db', {
    data: {
      overwrite: true,
      dashboard: {
        uid,
        title,
        schemaVersion: 41,
        time: { from: 'now-1h', to: 'now' },
        panels: [
          {
            id: 1,
            type: 'table',
            title: STREAM_PANEL_TITLE,
            gridPos: { h: 16, w: 24, x: 0, y: 0 },
            datasource,
            targets: [{ refId: 'A', datasource }],
            transformations: [{ id: 'sortBy', options: { sort: [{ field: TIMESTAMP_FIELD, desc: true }] } }],
            fieldConfig: { defaults: {}, overrides: [] },
            options: { showHeader: true },
          },
        ],
      },
    },
  });
  expect(res.ok(), `creating dashboard failed: ${await res.text()}`).toBeTruthy();
  return (await res.json()).uid;
}

export async function deleteDashboard(api: APIRequestContext, uid: string): Promise<void> {
  const res = await api.delete(`/api/dashboards/uid/${uid}`);
  expect([200, 404], await res.text()).toContain(res.status());
}

export async function deleteDatasource(api: APIRequestContext, uid: string): Promise<void> {
  const res = await api.delete(`/api/datasources/uid/${uid}`);
  expect([200, 404], await res.text()).toContain(res.status());
}

/** A short unique id usable in RabbitMQ object names and Grafana uids (max 40 chars). */
export function uniqueId(prefix = 'e2e'): string {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}
