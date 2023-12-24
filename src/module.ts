import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { RabbitMQQuery, RabbitMQDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, RabbitMQQuery, RabbitMQDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
