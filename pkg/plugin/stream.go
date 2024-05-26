package plugin

import (
	"context"
	"errors"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/maor2475/rabbitmq-datasource/pkg/rabbitmqclient"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func (ds *RabbitMQDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Info("Called RunStream method", "RabbitMQ Stream", ds.Client.ToString())

	framer := NewFramer()

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		log.DefaultLogger.Debug("Received message", "message", string(message.Data[0]))

		timestamped_msg := NewTimestampedMessage(message.Data[0])
		frame, err := framer.ToFrame(timestamped_msg)
		if err != nil {
			log.DefaultLogger.Error("Error creating frame from message", "message", string(message.Data[0]), "error", err)
			return
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Error sending frame because context canceled", "frame", frame, "error", err)
		default:
			err = sender.SendFrame(frame, data.IncludeAll)
			if err != nil {
				log.DefaultLogger.Error("Error sending frame", "frame", frame, "error", err)
			}
		}
	}

	for {
		log.DefaultLogger.Debug("Creating new consumer", "RabbitMQ Stream", ds.Client.ToString())
		if !ds.Client.IsConnected() {
			_, err := ds.Client.Connect()
			if err != nil {
				return err
			}
		}
		consumer, err := ds.Client.Consume(handleMessages)
		if errors.Is(err, rabbitmqclient.ErrConsumerWasAlreadyCreated) {
			return nil
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled", "RabbitMQ Stream", ds.Client.ToString())
			ds.Client.Dispose()
			return nil
		case <-consumer.NotifyClose():
			log.DefaultLogger.Info(
				"Something went wrong with the RabbitMQ. Trying to reconnect...",
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			ds.Client.Reconnect()
		}
	}
}

// SubscribeStream just returns an ok in this case, since we will always allow the user to successfully connect.
// Permissions verifications could be done here. Check backend.StreamHandler docs for more details.
func (ds *RabbitMQDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Info("Called SubscribeStream method", "RabbitMQ Stream", ds.Client.ToString())
	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

// PublishStream just returns permission denied in this case, since in this example we don't want the user to send stream data.
// Permissions verifications could be done here. Check backend.StreamHandler docs for more details.
func (ds *RabbitMQDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	log.DefaultLogger.Info("Called PublishStream method", "RabbitMQ Stream", ds.Client.ToString())
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
