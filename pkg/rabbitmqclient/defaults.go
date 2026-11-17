package rabbitmqclient

// Default data source settings. They are documented in README.md and must stay in sync with
// src/components/defaults.ts (the config editor) and provisioning/datasources/datasources.yml.
//
// The backend applies them itself because the settings saved by Grafana can be incomplete
// (e.g. a data source created via the API or provisioning with partial jsonData, or one saved
// from the config editor without touching any field).
const (
	DefaultHost                  = "localhost"
	DefaultAmqpPort              = 5672
	DefaultStreamPort            = 5552
	DefaultVHost                 = "/"
	DefaultUser                  = "guest"
	DefaultPassword              = "guest"
	DefaultRequestedHeartbeat    = 60 // seconds
	DefaultRequestedMaxFrameSize = 1048576
	DefaultWriteBuffer           = 8192
	DefaultReadBuffer            = 65536

	DefaultStreamName          = "rabbitmq.stream"
	DefaultMaxAge              = 3_600_000_000_000 // 1 hour, in nanoseconds
	DefaultMaxLengthBytes      = 2_000_000_000
	DefaultMaxSegmentSizeBytes = 500_000_000
)

// NewRabbitMQStreamOptions returns options pre-filled with the defaults. Unmarshalling the data
// source jsonData into the result only overrides the values that are present in the JSON.
func NewRabbitMQStreamOptions() *RabbitMQStreamOptions {
	return &RabbitMQStreamOptions{
		Host:                  DefaultHost,
		StreamPort:            DefaultStreamPort,
		AmqpPort:              DefaultAmqpPort,
		VHost:                 DefaultVHost,
		User:                  DefaultUser,
		Password:              DefaultPassword,
		RequestedHeartbeat:    DefaultRequestedHeartbeat,
		RequestedMaxFrameSize: DefaultRequestedMaxFrameSize,
		WriteBuffer:           DefaultWriteBuffer,
		ReadBuffer:            DefaultReadBuffer,
		StreamOptions:         NewStreamOptions(),
	}
}

// NewStreamOptions returns stream options pre-filled with the defaults.
func NewStreamOptions() *StreamOptions {
	return &StreamOptions{
		StreamName:          DefaultStreamName,
		MaxAge:              DefaultMaxAge,
		MaxLengthBytes:      DefaultMaxLengthBytes,
		MaxSegmentSizeBytes: DefaultMaxSegmentSizeBytes,
		OffsetFromStart:     true,
		ShouldDisposeStream: true,
	}
}
