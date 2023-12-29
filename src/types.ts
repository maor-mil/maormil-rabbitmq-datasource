import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface RabbitMQQuery extends DataQuery { }

export interface StreamOptions {
  streamName: string;
  maxAge: number;
  maxLengthBytes: number;
  maxSegmentSizeBytes: number;
  consumerName: string;
  
  crc: boolean;
}

export interface RabbitMQDataSourceOptions extends DataSourceJsonData {
  host: string;
  amqpPort: number;
  streamPort: number;
  vHost: string;

  tlsConnection?: boolean;
  username: string;

  streamOptions: StreamOptions;

  requestedHeartbeat: number;
  requestedMaxFrameSize: number;
  writeBuffer: number;
  readBuffer: number;
  noDelay: boolean;

  exchangeOptions: Exchanges;
  bindingOptions: Bindings;
}

export interface ExchangeOptions {
  name: string;
  type: string;
  durable: boolean;
  autoDeleted: boolean;
  internal: boolean;
  noWait: boolean;
}

export interface Exchanges extends Array<ExchangeOptions> {}

export interface BindingOptions {
  isQueueBinding: boolean;
  senderName: string;
  routingKey: string;
  receiverName: string;
  noWait: boolean;
}

export interface Bindings extends Array<BindingOptions> {}

export interface RabbitMQSecureJsonData {
  password?: string;
}
