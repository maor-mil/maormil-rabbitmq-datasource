import { test, expect } from './fixtures';
import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData } from '../src/types';

test.describe('provisioned data source', () => {
  test('"Save & test" succeeds and the provisioned topology exists in RabbitMQ', async ({
    gotoDataSourceConfigPage,
    readProvisionedDataSource,
    rabbitmq,
  }) => {
    const ds = await readProvisionedDataSource<RabbitMQDataSourceOptions, RabbitMQSecureJsonData>({
      fileName: 'datasources.yml',
    });
    const configPage = await gotoDataSourceConfigPage(ds.uid);
    await expect(configPage.saveAndTest()).toBeOK();

    const { streamOptions, exchangesOptions, bindingsOptions } = ds.jsonData;
    await expect.poll(() => rabbitmq.queueType(streamOptions.streamName)).toBe('stream');
    for (const exchange of exchangesOptions) {
      await expect.poll(() => rabbitmq.exchangeExists(exchange.name)).toBe(true);
    }
    for (const binding of bindingsOptions) {
      await expect
        .poll(async () =>
          (await rabbitmq.bindingsFrom(binding.senderName)).some(
            (b) => b.destination === binding.receiverName && b.routing_key === binding.routingKey
          )
        )
        .toBe(true);
    }
  });
});
