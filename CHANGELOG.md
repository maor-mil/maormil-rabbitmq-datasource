# Changelog

## Unreleased

### Breaking changes

* Minimum supported Grafana version is now **13.2.0** (was 9.4.3).

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
