import { test, expect } from '@grafana/plugin-e2e';
import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData } from '../src/types';

test('smoke: should render config editor', async ({ createDataSourceConfigPage, readProvisionedDataSource, page }) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await createDataSourceConfigPage({ type: ds.type });
  await expect(page.getByText('Stream Settings')).toBeVisible();
});

test('smoke: provisioned datasource "Save & test" should succeed against the local RabbitMQ', async ({
  gotoDataSourceConfigPage,
  readProvisionedDataSource,
}) => {
  const ds = await readProvisionedDataSource<RabbitMQDataSourceOptions, RabbitMQSecureJsonData>({
    fileName: 'datasources.yml',
  });
  const configPage = await gotoDataSourceConfigPage(ds.uid);
  await expect(configPage.saveAndTest()).toBeOK();
});
