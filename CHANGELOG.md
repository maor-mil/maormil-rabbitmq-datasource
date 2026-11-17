# Changelog

## Unreleased

### Breaking changes

* Minimum supported Grafana version is now **13.2.0** (was 9.4.3).
* The plugin **no longer deletes anything in RabbitMQ**. The `Should Dispose Stream`, `Should Dispose Exchange`, `Dispose if Unused` and `Should Dispose Binding` settings were removed (old saved values are ignored). They deleted the stream (with its stored messages), exchanges and bindings whenever the last panel was closed, and broke reconnecting after a stream was deleted in RabbitMQ.

### Fixed

* A `null` value after a non-null value crashed the plugin process (and every RabbitMQ data source with it), repeatedly, since the message was replayed after each restart. (#4, contributed by Henning Paulsen)
* A field whose JSON type changed between messages silently stopped receiving values. (#4)
* Streaming never resumed after the connection to RabbitMQ was lost; reconnecting is now driven by Grafana's stream retry with backoff. (#4)
* A stream deleted in RabbitMQ is recreated (with the configured exchanges and bindings) and streaming resumes, as documented.
* Reconnecting no longer duplicates the configured exchanges and bindings. (#4)
* A failure to create the consumer no longer panics (nil consumer). (#4)
* "Save & test" now shows the actual RabbitMQ connection error and reconnects an idle data source instead of reporting "RabbitMQ Disconnected".
* Messages are no longer dropped when panels cannot keep up (back-pressure instead of a full buffer); bodies sent as AMQP 1.0 `amqp-value` (binary or string) are supported.

### Added

* End-to-end test suite (Playwright + `@grafana/plugin-e2e`) running against Grafana and a real RabbitMQ broker: config editor, health check, provisioned topology, streaming (direct and through exchanges), and lifecycle/edge cases. It reproduces several existing bugs, which are tracked in `tests/known-bugs.ts`.
* The local/provisioned data source now declares an example fanout exchange (`rabbitmq.exchange`) bound to the stream.

### Changed

* Updated the plugin scaffolding from `@grafana/create-plugin` 4.11.1 to 9.0.3: React 19, `@grafana/*` 13.2.4, TypeScript 5.9, ESLint 9 flat config, Jest 30.
* Replaced the Cypress / `@grafana/e2e` setup (no longer supported by Grafana) with Playwright and `@grafana/plugin-e2e`.
* Updated the Go backend to Go 1.26 and `grafana-plugin-sdk-go` v0.299.0.
* The local `docker compose` environment now includes a RabbitMQ 4.2 broker with the stream plugin enabled.

## 1.0.0

* Initial release approved by Grafana Labs as a signed plugin.
