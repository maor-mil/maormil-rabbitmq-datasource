import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface RabbitMQQuery extends DataQuery { }

export interface StreamOptions {
  shouldDisposeStream: boolean;
  streamName: string;
  consumerName: string;
  offsetFromStart: boolean;
  maxAge: number;
  maxLengthBytes: number;
  maxSegmentSizeBytes: number;
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

  exchangesOptions: ExchangesOptions;
  bindingsOptions: BindingsOptions;

  requestedHeartbeat: number;
  requestedMaxFrameSize: number;
  writeBuffer: number;
  readBuffer: number;
  noDelay: boolean;
}

export interface ExchangeOptions {
  shouldDisposeExchange: boolean;
  disposeIfUnused: boolean;
  name: string;
  type: string;
  durable: boolean;
  autoDeleted: boolean;
  internal: boolean;
  noWait: boolean;
}

export interface ExchangesOptions extends Array<ExchangeOptions> {}

export interface BindingOptions {
  shouldDisposeBinding: boolean;
  isQueueBinding: boolean;
  senderName: string;
  routingKey: string;
  receiverName: string;
  noWait: boolean;
}

export interface BindingsOptions extends Array<BindingOptions> {}

export interface RabbitMQSecureJsonData {
  password?: string;
}
