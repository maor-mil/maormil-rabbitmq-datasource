import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig } from '@playwright/test';
import baseConfig from './.config/playwright.config';

/**
 * See https://playwright.dev/docs/test-configuration.
 * Extending the scaffolded config: https://grafana.com/developers/plugin-tools/how-to-guides/extend-configurations#extend-the-playwright-config
 */
export default defineConfig<PluginOptions>(baseConfig, {
  // Streaming tests share one RabbitMQ broker and provisioned datasource, so run spec files serially.
  fullyParallel: false,
  workers: 1,
  // Streaming assertions wait up to 30s for messages to reach a panel, on top of the dashboard and data source setup.
  // Tests that wait for Grafana Live / RabbitMQ lifecycle events raise this further with test.describe.configure().
  timeout: 90_000,
});
