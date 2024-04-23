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
	isFirstCtxDoneDispose := true

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		log.DefaultLogger.Debug("Received message", "message", string(message.Data[0]))

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
				if isFirstCtxDoneDispose {
					log.DefaultLogger.Debug("Stopped streaming - Context Canceled (in RabbitMQ Consumer handleMessages function)")
					ds.Client.Dispose()
					isFirstCtxDoneDispose = false
				}
				return
			default:
				log.DefaultLogger.Error("Error sending frame", "error", err)
			}
		}
	}

	for {
		log.DefaultLogger.Debug("Creating new consumer", "RabbitMQ Stream", ds.Client.ToString())
		err := ds.Client.Consume(handleMessages)
		if errors.Is(err, rabbitmqclient.ErrConsumerWasAlreadyCreated) {
			return nil
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled (in RunStream main for loop)")
			ds.Client.Dispose()
			return ctx.Err()
		default:
			log.DefaultLogger.Debug(
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
