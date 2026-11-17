---
name: e2e-test
description: Write, run and debug the Playwright/@grafana/plugin-e2e end-to-end tests of this plugin against Grafana and a real RabbitMQ broker. Use when adding a feature or bug fix that needs E2E coverage, when E2E fails locally or in CI, or when a known-bug test starts passing.
---

# E2E tests for the RabbitMQ data source

Read `.config/AGENTS/e2e-testing.md` (generic plugin-e2e rules) first. This skill adds the project specifics.

## Layout

| File | Purpose |
|---|---|
| `tests/fixtures.ts` | `test`/`expect` to import in every spec. Adds `rabbitmq` (management API client) and `createStreamingEnv()` |
| `tests/helpers/rabbitmq.ts` | Publish to a stream/exchange, inspect and delete queues, exchanges, bindings, users |
| `tests/helpers/grafana.ts` | Data source settings builder (`datasourceJsonData`), streaming dashboard builder, `uniqueId()` |
| `tests/known-bugs.ts` | `KNOWN_BUGS` + `expectKnownBug()` for tests that reproduce an open bug |
| `tests/0N-*.spec.ts` | Specs, run serially in file order (`workers: 1`); anything that can crash the plugin goes in the last file |

## Writing a test

```typescript
import { test, expect } from './fixtures';
import { uniqueId } from './helpers/grafana';

test('my streaming behaviour', async ({ createStreamingEnv, gotoDashboardPage, rabbitmq }) => {
  // Own data source + stream + dashboard (Table panel sorted newest-first); cleaned up after the test.
  const env = await createStreamingEnv({ streamOptions: { offsetFromStart: true } });
  const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
  const panel = dashboard.getPanelByTitle(env.panelTitle);
  await rabbitmq.waitForQueue(env.streamName); // the stream is declared when the data source instance starts

  const marker = uniqueId('msg');
  await rabbitmq.publishToStream(env.streamName, { name: marker, value: 1 });
  await expect(panel.locator.getByText(marker)).toBeVisible({ timeout: 30_000 });
});
```

Rules:
- Stream only through `createStreamingEnv`, never through the provisioned data source.
- Exchanges/bindings: pass `topology: ({ id, streamName }) => ({ exchangesOptions, bindingsOptions })`; name exchanges
  with the env `id` so the fixture deletes them.
- Per-panel sources: pass `panels: [{ title, query: { source: 'queue', queueMode: 'tap', exchange, routingKey } }]`
  (default: one `Stream` panel on the data source stream). Declare extra queues/exchanges with
  `rabbitmq.declareQueue/declareExchange/bindQueue` and delete them in a `finally`.
- Query editor tooltips (radio option descriptions) cover the row below after a click: `page.mouse.move(0, 0)`
  before clicking the next control.
- Use a unique marker per message and assert on it; the stream may replay older messages.
- Lifecycle assertions (Grafana Live stopping a stream, reconnects) need longer timeouts:
  `test.describe.configure({ timeout: 120_000 })`.
- Config editor fields are reachable by role and exact name, e.g.
  `page.getByRole('textbox', { name: 'Host', exact: true })`, `page.getByRole('switch', { name: 'CRC' })`.
- Hosts: the test runner talks to RabbitMQ on `localhost:15672`; data source settings must use `rabbitmq` (the
  compose service, `RABBITMQ_HOST_FROM_GRAFANA`).

## Known bugs

A test that reproduces an open bug calls `expectKnownBug(KNOWN_BUGS.someBug)` as its first statement, so it counts
as an expected failure. When you fix the bug, delete the `expectKnownBug` line and the `KNOWN_BUGS` entry; otherwise
the now-passing test is reported as an "unexpected pass" and fails the run. A test that times out never counts as
an expected failure, so keep assertion timeouts below the test timeout.

## Running

```bash
docker compose up -d --build --wait   # see the run-local-stack skill; rebuild dist/ first
npx playwright install chromium       # once
npm run e2e                           # all specs
npx playwright test tests/03-streaming.spec.ts -g "exchange"   # one spec / test
npx playwright test --ui              # interactive
PLAYWRIGHT_HTML_OPEN=never npx playwright test --reporter=list
```

## Debugging failures

- Local: `test-results/<test>/error-context.md` has the page snapshot; `npx playwright show-trace` for traces
  (recorded on retry in CI).
- Plugin side: `docker logs maormil-rabbitmq-datasource 2>&1 | grep plugin.maormil-rabbitmq-datasource`.
- CI: the failing matrix job uploads `e2e-<image>-v<version>-<run>` with the Playwright report and both containers'
  logs (`gh run download <run-id>`).
- `strict mode violation` usually means a non-exact role name (`Host` also matches `VHost`).
