import { CoreApp, DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';

import { RabbitMQQuery, RabbitMQDataSourceOptions } from './types';

export const DEFAULT_QUERY: Partial<RabbitMQQuery> = { source: 'datasource' };

export class DataSource extends DataSourceWithBackend<RabbitMQQuery, RabbitMQDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RabbitMQDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<RabbitMQQuery> {
    return DEFAULT_QUERY;
  }
}
