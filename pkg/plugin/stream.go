package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/maor2475/rabbitmq-datasource/pkg/rabbitmqclient"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func (ds *RabbitMQDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Info("RunStream Function was activated!")

	framer := NewFramer()

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		log.DefaultLogger.Debug(fmt.Sprintf("Message as string: %v", string(message.Data[0])))

		timestamped_msg := NewTimestampedMessage(message.Data[0])
		frame, err := framer.ToFrame(timestamped_msg)
		if err != nil {
			log.DefaultLogger.Error("Error creating frame", "error", err)
			return
		}

		err = sender.SendFrame(frame, data.IncludeAll)
		if err != nil {
			select {
			case <-ctx.Done():
				log.DefaultLogger.Info("Stopped streaming - Context Canceled (in RabbitMQ Consumer handleMessages function)")
				ds.Client.Dispose()
			default:
				log.DefaultLogger.Error("Error sending frame", "error", err)
			}
		}
	}

	for {
		log.DefaultLogger.Info(
			fmt.Sprintf("Creating new consumer for RabbitMQ %s",
				ds.Client.ToString(),
			),
		)
		err := ds.Client.Consume(handleMessages)
		if errors.Is(err, rabbitmqclient.ErrConsumerWasAlreadyCreated) {
			return nil
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("Stopped streaming - Context Canceled (in RunStream main for loop)")
			ds.Client.Dispose()
			return nil
		default:
			log.DefaultLogger.Info(
				fmt.Sprintf("Something went wrong with the RabbitMQ %s. Trying to reconnect...",
					ds.Client.ToString(),
				),
			)
			ds.Client.Reconnect()
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
