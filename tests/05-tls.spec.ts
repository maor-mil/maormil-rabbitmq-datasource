import { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { DS_TYPE, RABBITMQ_HOST_FROM_GRAFANA, uniqueId } from './helpers/grafana';
import { TLS_CLIENT_USER, TLS_PORTS, tlsFile } from './helpers/tls';

const STREAM_TIMEOUT = { timeout: 30_000 };
const textbox = (page: Page, name: string) => page.getByRole('textbox', { name, exact: true });
const enable = (page: Page, name: string) => page.getByRole('switch', { name, exact: true }).check({ force: true });

/** Opens a new data source pointed at the broker's TLS listeners, with the streaming name given. */
async function tlsConfigPage(
  createDataSourceConfigPage: Parameters<Parameters<typeof test>[2]>[0]['createDataSourceConfigPage'],
  page: Page,
  streamName: string
) {
  const configPage = await createDataSourceConfigPage({ type: DS_TYPE });
  await textbox(page, 'Host').fill(RABBITMQ_HOST_FROM_GRAFANA);
  await textbox(page, 'Stream Name').fill(streamName);
  await enable(page, 'TLS Connection');
  return configPage;
}

test.describe('TLS', () => {
  test('enabling TLS shows the TLS settings and switches to the TLS ports', async ({ createDataSourceConfigPage, page }) => {
    await createDataSourceConfigPage({ type: DS_TYPE });
    await expect(page.getByText('TLS Settings')).toHaveCount(0);

    await enable(page, 'TLS Connection');

    await expect(page.getByText('TLS Settings')).toBeVisible();
    await expect(textbox(page, 'AMQP Port')).toHaveValue(String(TLS_PORTS.amqpPort));
    await expect(textbox(page, 'Stream Port')).toHaveValue(String(TLS_PORTS.streamPort));
    await enable(page, 'With CA Cert');
    await expect(page.getByLabel('CA Cert', { exact: true })).toBeVisible();
  });

  test('"Save & test" succeeds over TLS when the broker is verified with its CA', async ({
    createDataSourceConfigPage,
    page,
    rabbitmq,
  }) => {
    const streamName = `${uniqueId()}.stream`;
    const configPage = await tlsConfigPage(createDataSourceConfigPage, page, streamName);
    await enable(page, 'With CA Cert');
    await page.getByLabel('CA Cert', { exact: true }).fill(tlsFile('ca.pem'));

    try {
      await expect(configPage.saveAndTest()).toBeOK();
      await expect(configPage).toHaveAlert('success', { hasText: 'RabbitMQ Connected' });
    } finally {
      await rabbitmq.deleteQueue(streamName);
    }
  });

  test('"Save & test" fails when the broker certificate is not signed by the configured CA', async ({
    createDataSourceConfigPage,
    page,
  }) => {
    const configPage = await tlsConfigPage(createDataSourceConfigPage, page, `${uniqueId()}.stream`);
    await enable(page, 'With CA Cert');
    await page.getByLabel('CA Cert', { exact: true }).fill(tlsFile('other-ca.pem'));

    await expect(configPage.saveAndTest()).not.toBeOK();
    await expect(configPage).toHaveAlert('error', { hasText: 'certificate' });
  });

  test('"Save & test" succeeds with mutual TLS and EXTERNAL (client certificate) authentication', async ({
    createDataSourceConfigPage,
    page,
    rabbitmq,
  }) => {
    // EXTERNAL maps the certificate CN to a RabbitMQ user, which must exist; its password is never used.
    await rabbitmq.createUser(TLS_CLIENT_USER, uniqueId('unused-password'));
    const streamName = `${uniqueId()}.stream`;
    const configPage = await tlsConfigPage(createDataSourceConfigPage, page, streamName);
    await page.getByRole('radio', { name: 'Client certificate' }).check({ force: true });
    await expect(textbox(page, 'Username')).toHaveCount(0);
    await enable(page, 'With CA Cert');
    await page.getByLabel('CA Cert', { exact: true }).fill(tlsFile('ca.pem'));
    await enable(page, 'TLS Client Auth');
    await page.getByLabel('Client Cert', { exact: true }).fill(tlsFile('client.pem'));
    await page.getByLabel('Client Key', { exact: true }).fill(tlsFile('client-key.pem'));

    try {
      await expect(configPage.saveAndTest()).toBeOK();
      await expect(configPage).toHaveAlert('success', { hasText: 'RabbitMQ Connected' });
    } finally {
      await rabbitmq.deleteQueue(streamName);
      await rabbitmq.deleteUser(TLS_CLIENT_USER);
    }
  });

  test('messages stream to a panel over TLS', async ({ createStreamingEnv, gotoDashboardPage, rabbitmq }) => {
    const env = await createStreamingEnv({
      connection: { tlsConnection: true, ...TLS_PORTS, tlsAuthWithCACert: true },
      secureJsonData: { password: 'guest', tlsCACert: tlsFile('ca.pem') },
    });
    const dashboard = await gotoDashboardPage({ uid: env.dashboardUid });
    const panel = dashboard.getPanelByTitle(env.panelTitle);
    await rabbitmq.waitForQueue(env.streamName);

    const marker = uniqueId('over-tls');
    await rabbitmq.publishToStream(env.streamName, { name: marker });
    await expect(panel.locator.getByText(marker)).toBeVisible(STREAM_TIMEOUT);
  });
});
