package rabbitmqclient

import (
	"errors"
	"fmt"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

var ErrConsumerWasAlreadyCreated = errors.New("consumer was already created")

type Stream interface {
	CreateStream(*stream.Environment) error
	DisposeStream(*stream.Environment) error
	Consume(*stream.Environment, stream.MessagesHandler) (*stream.Consumer, error)
}

type StreamOptions struct {
	StreamName          string        `json:"streamName"`
	MaxAge              time.Duration `json:"maxAge"`
	MaxLengthBytes      int64         `json:"maxLengthBytes"`
	MaxSegmentSizeBytes int64         `json:"maxSegmentSizeBytes"`
	ConsumerName        string        `json:"consumerName"`
	OffsetFromStart     bool          `json:"offsetFromStart"`
	Crc                 bool          `json:"crc"`
	ShouldDisposeStream bool          `json:"ShouldDisposeStream"`
	Consumer            *stream.Consumer
}

func (streamOptions *StreamOptions) CreateStream(env *stream.Environment) error {
	err := env.DeclareStream(streamOptions.StreamName,
		stream.NewStreamOptions().
			SetMaxAge(streamOptions.MaxAge*time.Nanosecond).
			SetMaxLengthBytes(stream.ByteCapacity{}.B(streamOptions.MaxLengthBytes)).
			SetMaxSegmentSizeBytes(stream.ByteCapacity{}.B(streamOptions.MaxSegmentSizeBytes)))

	return err
}

func (streamOptions *StreamOptions) DisposeStream(env *stream.Environment) error {
	err := streamOptions.closeConsumer()
	if err != nil {
		return err
	}

	if streamOptions.ShouldDisposeStream {
		return env.DeleteStream(streamOptions.StreamName)
	}

	return nil
}

func (streamOptions *StreamOptions) Consume(env *stream.Environment, messagesHandler stream.MessagesHandler) (*stream.Consumer, error) {
	if streamOptions.Consumer != nil {
		return nil, failOnError(ErrConsumerWasAlreadyCreated,
			fmt.Sprintf("StreamName: %s; ConsumerName:%s",
				streamOptions.ConsumerName,
				streamOptions.getConsumerName(),
			),
		)
	}
	consumer, err := env.NewConsumer(
		streamOptions.StreamName,
		messagesHandler,
		stream.NewConsumerOptions().
			SetConsumerName(streamOptions.getConsumerName()). // Set a consumer name
			SetOffset(streamOptions.getOffsetSettings()).     // Start consuming from the beginning
			SetCRCCheck(streamOptions.Crc),                   // Disabled CRC control increase the performances
	)
	if err != nil {
		return nil, failOnError(err, fmt.Sprintf("Failed to create the consumer: %s", streamOptions.ConsumerName))
	}
	streamOptions.Consumer = consumer
	return streamOptions.Consumer, nil
}

func (streamOptions *StreamOptions) getOffsetSettings() stream.OffsetSpecification {
	offsetSettings := stream.OffsetSpecification{}
	if streamOptions.OffsetFromStart {
		offsetSettings = offsetSettings.First()
	} else {
		offsetSettings = offsetSettings.Last()
	}
	return offsetSettings
}

func (streamOptions *StreamOptions) getConsumerName() string {
	if streamOptions.ConsumerName == "" {
		return fmt.Sprintf("%s_consumer", streamOptions.StreamName)
	}
	return streamOptions.ConsumerName
}

func (streamOptions *StreamOptions) closeConsumer() error {
	if streamOptions.Consumer == nil {
		return nil
	} else if err := streamOptions.Consumer.Close(); err != nil {
		return failOnError(err, fmt.Sprintf("Failed to close the consumer: %s", streamOptions.ConsumerName))
	} else {
		streamOptions.Consumer = nil
		return nil
	}
}
