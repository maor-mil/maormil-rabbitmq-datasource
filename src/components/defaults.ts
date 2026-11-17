import { RabbitMQDataSourceOptions, StreamOptions } from '../types';

// Default values of the data source settings. They are documented in README.md and mirrored in
// pkg/rabbitmqclient/defaults.go and provisioning/datasources/datasources.yml — keep them in sync.

export const DEFAULT_PASSWORD = 'guest';

export const DEFAULT_STREAM_OPTIONS: StreamOptions = {
  shouldDisposeStream: true,
  streamName: 'rabbitmq.stream',
  consumerName: '',
  offsetFromStart: true,
  maxAge: 1_000_000_000 * 60 * 60, // 1 hour, in nanoseconds
  maxLengthBytes: 2_000_000_000,
  maxSegmentSizeBytes: 500_000_000,
  crc: false,
};

export const DEFAULT_OPTIONS: RabbitMQDataSourceOptions = {
  host: 'localhost',
  amqpPort: 5672,
  streamPort: 5552,
  vHost: '/',
  tlsConnection: false,
  username: 'guest',
  streamOptions: DEFAULT_STREAM_OPTIONS,
  exchangesOptions: [],
  bindingsOptions: [],
  requestedHeartbeat: 60,
  requestedMaxFrameSize: 1048576,
  writeBuffer: 8192,
  readBuffer: 65536,
  noDelay: false,
};

/** Returns the settings with every missing value (including nested stream options) filled with its default. */
export function withDefaults(jsonData: Partial<RabbitMQDataSourceOptions> | undefined): RabbitMQDataSourceOptions {
  const current = jsonData ?? {};
  return {
    ...DEFAULT_OPTIONS,
    ...current,
    streamOptions: { ...DEFAULT_STREAM_OPTIONS, ...current.streamOptions },
    exchangesOptions: current.exchangesOptions ?? [],
    bindingsOptions: current.bindingsOptions ?? [],
  };
}

