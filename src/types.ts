import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

/**
 * What a panel streams:
 * - datasource: the stream configured on the data source (default; the only option before 2.0)
 * - stream: any existing stream, read with the stream protocol (RabbitMQ 3.9+)
 * - queue: classic/quorum queues over AMQP 1.0 (RabbitMQ 4.0+)
 */
export type SourceType = 'datasource' | 'stream' | 'queue';
export type StreamOffset = 'first' | 'last' | 'next';
/** tap: a temporary queue bound to an exchange receives copies; existing: consume (and remove) messages of a queue. */
export type QueueMode = 'tap' | 'existing';

export interface RabbitMQQuery extends DataQuery {
  source?: SourceType;
  /** source=stream */
  stream?: string;
  offset?: StreamOffset;
  /** Bloom filter values (RabbitMQ 3.13+); matchUnfiltered also delivers messages published without a value. */
  filterValues?: string[];
  matchUnfiltered?: boolean;
  /** SQL filter expression evaluated by RabbitMQ for every message (RabbitMQ 4.2+, read over AMQP 1.0). */
  sql?: string;
  /** source=queue */
  queueMode?: QueueMode;
  exchange?: string;
  routingKey?: string;
  queue?: string;
}

export interface StreamOptions {
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
  /** PLAIN = username/password, EXTERNAL = the TLS client certificate (requires TLS Client Auth). */
  saslMechanism: SaslMechanism;
  username: string;

  // TLS settings, using Grafana's standard keys. Only used when tlsConnection is on.
  /** Present a client certificate (mutual TLS). */
  tlsAuth: boolean;
  /** Verify the broker with a custom CA certificate instead of the system roots. */
  tlsAuthWithCACert: boolean;
  tlsSkipVerify: boolean;
  /** Overrides the host name used to verify the broker certificate (SNI). */
  serverName: string;
  tlsMinVersion: TlsMinVersion;

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
  name: string;
  type: string;
  durable: boolean;
  autoDeleted: boolean;
  internal: boolean;
  noWait: boolean;
}

export interface ExchangesOptions extends Array<ExchangeOptions> {}

export interface BindingOptions {
  isQueueBinding: boolean;
  senderName: string;
  routingKey: string;
  receiverName: string;
  noWait: boolean;
}

export interface BindingsOptions extends Array<BindingOptions> {}

export type SaslMechanism = 'PLAIN' | 'EXTERNAL';
export type TlsMinVersion = '1.2' | '1.3';

export interface RabbitMQSecureJsonData {
  password?: string;
  /** PEM CA certificate (with tlsAuthWithCACert). */
  tlsCACert?: string;
  /** PEM client certificate and key (with tlsAuth). */
  tlsClientCert?: string;
  tlsClientKey?: string;
}
