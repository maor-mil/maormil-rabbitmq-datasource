# RabbitMQ Grafana data source (`maormil-rabbitmq-datasource`)

This repository contains a **Grafana plugin**. You must Read @./.config/AGENTS/instructions.md before doing changes
(Grafana's own agent rules; your training data about Grafana APIs is out of date — fetch
https://grafana.com/developers/plugin-tools/llms.txt and append `.md` to any docs URL for plain text).

A **backend streaming data source**, signed and published in the Grafana plugin catalog. The Go backend consumes a
RabbitMQ stream and pushes every JSON message to panels as a one-row data frame over **Grafana Live**. There is no
query language: everything is configured on the data source.

## Architecture

Data flow for a panel:

1. `QueryData` (`pkg/plugin/query.go`) parses the panel query (`RabbitMQQuery`: source `datasource` | `stream` |
   `queue`) into a normalised `rabbitmqclient.Source` and returns an empty frame whose `meta.channel` encodes it
   (`pkg/plugin/source_path.go`): `ds/<uid>/rabbitmq` for the data source stream (unchanged since 1.x),
   `ds/<uid>/src/<base64url JSON>` otherwise, or `ref/<hash>` (in-process registry) when that exceeds Live's
   160-character channel limit. Equal sources → same channel → one shared consumer.
2. The frontend subscribes to that Live channel → `SubscribeStream` (validates the path) → Grafana runs **one**
   `RunStream` per channel while it has subscribers.
3. `RunStream` (`pkg/plugin/stream.go`) dispatches on the source: the data source stream (stream protocol, single
   named consumer), any other stream (`ConsumeStream`, anonymous consumer on the shared environment) or a queue
   (`ConsumeQueue` in `pkg/rabbitmqclient/amqp10.go`: its own AMQP 1.0 connection per channel via `Azure/go-amqp`;
   tap mode first declares a server-named auto-delete queue with `x-expires` over AMQP 0.9.1). A stream source
   with an SQL filter is read over AMQP 1.0 too (`pkg/rabbitmqclient/stream_filter.go`); Bloom filter values alone
   use the stream protocol (`ConsumeStream` + post-filter on the `x-stream-filter-value` annotation).
   **AMQP 1.0 brokers silently drop link filters they don't accept** — always check
   `receiver.LinkSourceFilterValue(...)` after attaching (`openFilteredStream` does), or a bad SQL expression
   streams everything unfiltered. Don't use
   `rabbitmq-amqp-go-client`: v1.6.0 does not compile for 32-bit targets (linux/arm), which `mage buildAll` builds. For each message
   `Framer.ToFrame` (`pkg/plugin/framer.go`) flattens the JSON object into fields, adds the `RmqMsgConsumedTimestamp`
   time field and calls `sender.SendFrame`.
4. When the channel has no subscribers left, Grafana Live cancels `RunStream`'s context; `RunStream` then closes
   only its own consumer (other panels share the connection). If the consumer dies (connection loss, stream deleted) `RunStream` closes and
   **returns an error**, and Grafana re-runs it with backoff — reconnecting is Grafana's job, not a plugin loop.

Instance lifecycle (`pkg/plugin/datasource.go`): `NewRabbitMQInstance` runs on the first request for a data source
(and again after **every settings save** — instancemgmt then calls `Dispose()` on the old instance; there is no
"data source deleted" hook). It parses settings (`getDatasourceSettings`) and tries to `Connect()` (a failure is only
logged; `CheckHealth` reconnects and reports the error). `Connect()` declares exchanges → the stream → bindings over
a short-lived AMQP 0.9.1 connection and keeps a stream-protocol `Environment`; declaring is idempotent, so it also
recreates a stream deleted in RabbitMQ.

**The plugin never deletes anything in RabbitMQ** (streams, exchanges, bindings): `Client.Close()` only releases the
consumer and connections. Don't add deletion to stop/dispose paths — they run on dashboard close and settings save.

| Path | What |
|---|---|
| `pkg/plugin/` | SDK handlers: datasource (instance), query, stream (Live), health, framer |
| `pkg/rabbitmqclient/` | `client.go` (connect/declare/close, mutex-guarded), `stream.go` (declare/consume), `exchange.go`, `binding.go`, `defaults.go` |
| `src/` | `components/ConfigEditor.tsx` (all settings), `QueryEditor.tsx` (stub), `types.ts`, `components/defaults.ts` |
| `tests/` | Playwright E2E suite (see below) |
| `provisioning/` | Data source + dashboard loaded by the dev/E2E Grafana |
| `docker/rabbitmq/` | Broker config for the compose stack (`enabled_plugins`, TLS listeners in `20-tls.conf`); `tls/generate-certs.sh` creates throwaway test certificates in the git-ignored `tls/generated/` via the one-shot `rabbitmq-certs` service (read them in tests with `tests/helpers/tls.ts`) |
| `demo/` | `go run ./demo` publishes random JSON into `rabbitmq.stream` |

## Settings contract (keep in sync)

Settings live in `jsonData`; secrets (`password`, `tlsCACert`, `tlsClientCert`, `tlsClientKey`) are in `secureJsonData`.
TLS uses Grafana's standard keys (`tlsAuth`, `tlsAuthWithCACert`, `tlsSkipVerify`, `serverName`) plus `tlsMinVersion` and
`saslMechanism` (`PLAIN`/`EXTERNAL`); `pkg/rabbitmqclient/tls.go` builds one `*tls.Config` for both connections. The shape and defaults are defined in **four**
places that must change together — use the `add-datasource-setting` skill:

1. `src/types.ts` and `src/components/defaults.ts` (UI)
2. Go struct json tags in `pkg/rabbitmqclient` (`RabbitMQStreamOptions`, `StreamOptions`, `ExchangeOptions`,
   `BindingOptions`) and `pkg/rabbitmqclient/defaults.go`
3. `provisioning/datasources/datasources.yml`
4. The settings tables in `README.md`

Gotchas:
- The backend applies defaults to missing fields (unmarshal into pre-filled structs). Grafana 13 drops
  `onOptionsChange` calls made while the config page initialises, so the editor must never rely on persisting
  defaults on mount; every change writes the **complete** `jsonData` in a **single** `onOptionsChange`.
- Go json matching is case-insensitive; unknown keys are ignored (old saved settings may still contain the removed
  `shouldDispose*` keys — the editor strips them on save).
- Units: `requestedHeartbeat` is seconds, `maxAge` is nanoseconds.

## Commands

Prerequisites: Node 24 (`.nvmrc`), Go from `go.mod` (the toolchain downloads it), [mage](https://magefile.org), Docker.

| Task | Command |
|---|---|
| Install / build frontend | `npm install`, `npm run dev` (watch) or `npm run build` |
| Build backend for the container | `mage -v build:linux` → `dist/gpx_rabbitmq_linux_amd64` (all platforms: `mage -v buildAll`) |
| Checks | `npm run typecheck && npm run lint && npm run test:ci`; `go test ./pkg/...` or `mage coverage` |
| Go integration tests (real broker) | `docker compose up -d --wait rabbitmq && go test -tags integration ./pkg/...` |
| Go lint (same as CI) | `docker run --rm -v "$PWD:/app" -w /app golangci/golangci-lint:latest golangci-lint run ./...` |
| Run Grafana + RabbitMQ | `npm run server` → Grafana http://localhost:3000 (anonymous admin), RabbitMQ UI http://localhost:15672 (guest/guest) |
| After rebuilding the backend | `docker compose restart grafana` (frontend changes only need a browser reload) |
| E2E | `npx playwright install chromium` once, then `npm run e2e` with the stack running |

Skills: `run-local-stack`, `e2e-test`, `add-datasource-setting`, `update-create-plugin`, `release-plugin`, and the
create-plugin ones `build-plugin`, `validate-plugin` (`.claude/skills/`).

Windows: the repo path contains spaces — always quote paths. The `@grafana/create-plugin` CLI refuses to run on
Windows (use a `node:24` container, see `update-create-plugin`). In Git Bash, prefix `docker run -v` commands with
`MSYS_NO_PATHCONV=1` and use `$(pwd -W)`. `.gitattributes` keeps LF for files used inside Linux containers.

## CI (`.github/workflows/`)

| Workflow | Runs on | Does |
|---|---|---|
| `ci.yml` | push to `main`, **any** pull request (any base branch), manual | typecheck, lint, Jest, build, golangci-lint, `mage buildAll` + `coverage`, plugin-validator metadata check → Playwright E2E matrix resolved from `grafanaDependency` (min version … latest + nightly) against the compose stack |
| `is-compatible.yml` | pull requests | API compatibility with the latest `@grafana/*` |
| `release.yml` | `v*` tags | `grafana/plugin-actions/build-plugin`, signs with the `POLICY_TOKEN` secret, creates a draft release; the tag must equal `v` + `package.json` version |

A branch push without a PR runs nothing (use `workflow_dispatch`). On E2E failure CI uploads the Playwright report
and both containers' logs as an artifact.

## E2E tests (`tests/`)

- Read `.config/AGENTS/e2e-testing.md` (plugin-e2e rules) first; the `e2e-test` skill has the project specifics.
- Anything that streams or touches RabbitMQ topology uses the `createStreamingEnv` fixture (`tests/fixtures.ts`):
  an isolated data source + stream + streaming Table dashboard per test, removed afterwards. Never stream through
  the provisioned data source: a bad message would be replayed into every later test.
- Publish and inspect the broker with the `rabbitmq` fixture (`tests/helpers/rabbitmq.ts`, management HTTP API).
  The test runner reaches RabbitMQ at `localhost`; Grafana reaches it at `rabbitmq` (compose service name).
- Spec files are numbered and run serially (`workers: 1`); tests that can crash the plugin process go last.
- `tests/known-bugs.ts` lists bugs the suite reproduces; `expectKnownBug(...)` marks those tests as expected
  failures. **When you fix one, remove its marker** — the test otherwise fails as an "unexpected pass".

## RabbitMQ facts that matter here

- Ports: stream 5552 (TLS 5551), AMQP 5672 (TLS 5671), management 15672. amqp091-go only uses TLS with the `amqps`
  scheme. SASL EXTERNAL maps the client certificate CN to an existing RabbitMQ user.
- Version gates: streams 3.9+; stream filtering (Bloom) 3.13+ and stream client ≥ v1.3.3; the AMQP 1.0 Go client
  (`rabbitmq-amqp-go-client`) needs RabbitMQ 4.0+; property filters 4.1+; **SQL filter expressions 4.2+ and only over
  AMQP 1.0** (not the stream protocol).
- Messages published with AMQP 0.9.1 into a stream arrive as one AMQP 1.0 `Data` section; non-`x-` headers become
  application properties, `x-` headers become message annotations.
- Consuming a classic/quorum queue removes the messages (competing consumers); streams are non-destructive.

## Grafana plugin rules (validator / review)

- Secrets only in `secureJsonData`; never log credentials at info level or above.
- Backend: no filesystem or environment access, no `fmt.Print*` (use `backend/log`), always close connections,
  no `panic`. Accept certificates as PEM content, not file paths. Mark an intentional `InsecureSkipVerify` with
  `// #nosec G402` and a reason (gosec runs in the validator).
- Do not change the plugin `id`/`type`; `plugin.json` changes need a Grafana restart.
- Do not edit `.config/` (managed by create-plugin); extend configs from the root files.
- Keep `CHANGELOG.md` (`## Unreleased`) and the README settings tables up to date with user-visible changes.

## Useful docs

- Backend plugins: https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins
- Streaming data sources (RunStream / SubscribeStream): https://grafana.com/developers/plugin-tools/tutorials/build-a-streaming-data-source-plugin
- Auth & secrets: https://grafana.com/developers/plugin-tools/how-to-guides/data-source-plugins/add-authentication-for-data-source-plugins
- plugin.json reference: https://grafana.com/developers/plugin-tools/reference/plugin-json
- E2E: https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/get-started
- Signing / publishing: https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin
- Plugin validator: https://github.com/grafana/plugin-validator
- RabbitMQ streams: https://www.rabbitmq.com/docs/streams, filtering: https://www.rabbitmq.com/docs/stream-filtering
- Optional, more Grafana skills: `claude plugin marketplace add grafana/skills` (e.g. `react-19-plugin-migration`).
