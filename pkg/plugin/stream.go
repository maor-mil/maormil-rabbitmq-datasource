package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func (ds *RabbitMQDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Info("RunStream Function was activated!")

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		log.DefaultLogger.Info("Full Byte Array: %s", message)
		for i := 0; i < len(message.Data); i += 1 {
			log.DefaultLogger.Info("Byte Array At %d: %s", i, message.Data[i])
		}
	}

	log.DefaultLogger.Info("Trying to consume!")
	ds.Client.Consume(handleMessages)
	log.DefaultLogger.Info("Consumer was created!")

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("stopped streaming (context canceled)")
			ds.Client.CloseConsumers()
			return nil
		default:
			// log.DefaultLogger.Info("Shit is happening")
		}
	}
}

func (ds *RabbitMQDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Info("SubscribeStream Function was activated!")
	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

func (ds *RabbitMQDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	log.DefaultLogger.Info("PublishStream Function was activated!")
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
