import { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { DS_TYPE, RABBITMQ_HOST_FROM_GRAFANA, uniqueId } from './helpers/grafana';

const textbox = (page: Page, name: string) => page.getByRole('textbox', { name, exact: true });

test.describe('config editor', () => {
  test('renders the documented default values for a new data source', async ({ createDataSourceConfigPage, page }) => {
    await createDataSourceConfigPage({ type: DS_TYPE });

    await expect(page.getByRole('group', { name: 'Connection', exact: true })).toBeVisible();
    await expect(textbox(page, 'Host')).toHaveValue('localhost');
    await expect(textbox(page, 'AMQP Port')).toHaveValue('5672');
    await expect(textbox(page, 'Stream Port')).toHaveValue('5552');
    await expect(textbox(page, 'VHost')).toHaveValue('/');
    await expect(textbox(page, 'Username')).toHaveValue('guest');
    await expect(textbox(page, 'Stream Name')).toHaveValue('rabbitmq.stream');
    await expect(page.getByRole('switch', { name: 'TLS Connection' })).not.toBeChecked();
    await expect(page.getByRole('switch', { name: 'Offset From Start' })).toBeChecked();
  });

  test('"Save & test" succeeds and declares the stream when the settings are valid', async ({
    createDataSourceConfigPage,
    page,
    rabbitmq,
  }) => {
    const streamName = `${uniqueId()}.stream`;
    const configPage = await createDataSourceConfigPage({ type: DS_TYPE });
    await textbox(page, 'Host').fill(RABBITMQ_HOST_FROM_GRAFANA);
    await textbox(page, 'Stream Name').fill(streamName);

    try {
      await expect(configPage.saveAndTest()).toBeOK();
      await expect(configPage).toHaveAlert('success', { hasText: 'RabbitMQ Connected' });
      await expect.poll(() => rabbitmq.queueType(streamName)).toBe('stream');
    } finally {
      await rabbitmq.deleteQueue(streamName);
    }
  });

  test('"Save & test" fails when the broker cannot be reached', async ({ createDataSourceConfigPage, page }) => {
    const configPage = await createDataSourceConfigPage({ type: DS_TYPE });
    await textbox(page, 'Host').fill(RABBITMQ_HOST_FROM_GRAFANA);
    await textbox(page, 'Stream Port').fill('5553');

    await expect(configPage.saveAndTest()).not.toBeOK();
    await expect(configPage).toHaveAlert('error', { hasText: 'Cannot connect to RabbitMQ' });
  });

  test('"Save & test" fails when the credentials are wrong', async ({ createDataSourceConfigPage, page }) => {
    const configPage = await createDataSourceConfigPage({ type: DS_TYPE });
    await textbox(page, 'Host').fill(RABBITMQ_HOST_FROM_GRAFANA);
    await textbox(page, 'Username').fill('no-such-user');

    await expect(configPage.saveAndTest()).not.toBeOK();
    await expect(configPage).toHaveAlert('error', { hasText: 'Cannot connect to RabbitMQ' });
  });
});
