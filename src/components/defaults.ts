import { RabbitMQDataSourceOptions, StreamOptions } from '../types';

// Default values of the data source settings. They are documented in README.md and mirrored in
// pkg/rabbitmqclient/defaults.go and provisioning/datasources/datasources.yml — keep them in sync.

export const DEFAULT_PASSWORD = 'guest';

export const DEFAULT_STREAM_OPTIONS: StreamOptions = {
  streamName: 'rabbitmq.stream',
  consumerName: '',
  offsetFromStart: true,
  maxAge: 1_000_000_000 * 60 * 60, // 1 hour, in nanoseconds
  maxLengthBytes: 2_000_000_000,
  maxSegmentSizeBytes: 500_000_000,
  crc: false,
};

/** Standard RabbitMQ ports without and with TLS. */
export const PLAIN_PORTS = { amqpPort: 5672, streamPort: 5552 };
export const TLS_PORTS = { amqpPort: 5671, streamPort: 5551 };

export const DEFAULT_OPTIONS: RabbitMQDataSourceOptions = {
  host: 'localhost',
  amqpPort: 5672,
  streamPort: 5552,
  vHost: '/',
  tlsConnection: false,
  saslMechanism: 'PLAIN',
  username: 'guest',
  tlsAuth: false,
  tlsAuthWithCACert: false,
  tlsSkipVerify: false,
  serverName: '',
  tlsMinVersion: '1.2',
  streamOptions: DEFAULT_STREAM_OPTIONS,
  exchangesOptions: [],
  bindingsOptions: [],
  requestedHeartbeat: 60,
  requestedMaxFrameSize: 1048576,
  writeBuffer: 8192,
  readBuffer: 65536,
  noDelay: false,
};

/**
 * Settings that older plugin versions stored but that no longer exist (the plugin stopped deleting
 * RabbitMQ objects in 2.0). They are dropped from the settings the editor writes back.
 */
const REMOVED_STREAM_KEYS = ['shouldDisposeStream'];
const REMOVED_EXCHANGE_KEYS = ['shouldDisposeExchange', 'disposeIfUnused'];
const REMOVED_BINDING_KEYS = ['shouldDisposeBinding'];

function without<T extends object>(value: T, keys: string[]): T {
  const copy = { ...value } as Record<string, unknown>;
  keys.forEach((key) => delete copy[key]);
  return copy as T;
}

/** Returns the settings with every missing value (including nested stream options) filled with its default. */
export function withDefaults(jsonData: Partial<RabbitMQDataSourceOptions> | undefined): RabbitMQDataSourceOptions {
  const current = jsonData ?? {};
  return {
    ...DEFAULT_OPTIONS,
    ...current,
    streamOptions: without({ ...DEFAULT_STREAM_OPTIONS, ...current.streamOptions }, REMOVED_STREAM_KEYS),
    exchangesOptions: (current.exchangesOptions ?? []).map((e) => without(e, REMOVED_EXCHANGE_KEYS)),
    bindingsOptions: (current.bindingsOptions ?? []).map((b) => without(b, REMOVED_BINDING_KEYS)),
  };
}

