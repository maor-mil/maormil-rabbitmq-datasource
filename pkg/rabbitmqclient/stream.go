package rabbitmqclient

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

type Stream interface {
	CreateStream(env *stream.Environment) error
	Consume(*stream.Environment, stream.MessagesHandler) (*stream.Consumer, error)
	CloseConsumers() error
}

type StreamOptions struct {
	StreamName          string        `json:"streamName"`
	MaxAge              time.Duration `json:"maxAge"`
	MaxLengthBytes      int64         `json:"maxLengthBytes"`
	MaxSegmentSizeBytes int64         `json:"maxSegmentSizeBytes"`
	Crc                 bool          `json:"crc"`
	Consumers           []*stream.Consumer
}

func NewStreamOptions() *StreamOptions {
	return &StreamOptions{}
}

func (streamOptions *StreamOptions) CreateStream(env *stream.Environment) error {
	log.DefaultLogger.Info(fmt.Sprintf("env: %+v: ", env))
	err := env.DeclareStream(streamOptions.StreamName,
		stream.NewStreamOptions().
			SetMaxAge(streamOptions.MaxAge).
			SetMaxLengthBytes(stream.ByteCapacity{}.B(streamOptions.MaxLengthBytes)).
			SetMaxSegmentSizeBytes(stream.ByteCapacity{}.B(streamOptions.MaxSegmentSizeBytes)))

	return err
}

func (streamOptions *StreamOptions) Consume(env *stream.Environment, messagesHandler stream.MessagesHandler) (*stream.Consumer, error) {
	consumer, err := env.NewConsumer(
		streamOptions.StreamName,
		messagesHandler,
		stream.NewConsumerOptions().
			SetConsumerName("my_consumer").                  // set a consumer name
			SetOffset(stream.OffsetSpecification{}.First()). // start consuming from the beginning
			SetCRCCheck(streamOptions.Crc),                  // Disabled CRC control increase the performances
	)
	if err != nil {
		return nil, err
	}
	streamOptions.Consumers = append(streamOptions.Consumers, consumer)
	defer consumerClose(consumer.NotifyClose())
	return consumer, nil
}

func consumerClose(channelClose stream.ChannelClose) {
	event := <-channelClose
	log.DefaultLogger.Info(fmt.Sprintf("Consumer: %s closed on the stream: %s, reason: %s \n", event.Name, event.StreamName, event.Reason))
}

func (streamOptions *StreamOptions) CloseConsumers() error {
	for consumerIndex := 0; consumerIndex < len(streamOptions.Consumers); consumerIndex += 1 {
		if err := streamOptions.Consumers[consumerIndex].Close(); err != nil {
			return err
		}
	}
	return nil
}
