import {
  DataSourceInstanceSettings,
} from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';

import { RabbitMQQuery, RabbitMQDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<RabbitMQQuery, RabbitMQDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RabbitMQDataSourceOptions>) {
    super(instanceSettings);
  }
}
