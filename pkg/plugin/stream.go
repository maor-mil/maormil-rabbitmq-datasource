package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func (ds *RabbitMQDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Info("Called RunStream method", "RabbitMQ Stream", ds.Client.ToString())

	framer := NewFramer()
	msgCh := make(chan []byte, 256)

	// Sender goroutine: decouples frame building and SendFrame from the RabbitMQ
	// consumer callback, preventing back-pressure from blocking message delivery.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-msgCh:
				if !ok {
					return
				}
				log.DefaultLogger.Debug("Received message", "message", string(raw))
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.DefaultLogger.Error("Panic processing message, skipping",
								"recover", fmt.Sprintf("%v", r),
								"RabbitMQ Stream", ds.Client.ToString(),
							)
						}
					}()
					timestamped_msg := NewTimestampedMessage(raw)
					frame, err := framer.ToFrame(timestamped_msg)
					if err != nil {
						log.DefaultLogger.Error("Error creating frame from message", "error", err)
						return
					}
					if err = sender.SendFrame(frame, data.IncludeAll); err != nil {
						log.DefaultLogger.Error("Error sending frame", "error", err)
					}
				}()
			}
		}
	}()

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		var raw []byte
		if len(message.Data) > 0 {
			// AMQP 1.0 Data section (most stream-native publishers).
			raw = make([]byte, len(message.Data[0]))
			copy(raw, message.Data[0])
		} else if v, ok := message.Value.([]byte); ok {
			// AMQP Value section carrying a binary payload (common when messages
			// are published via an AMQP 0-9-1 client that is routed into a stream).
			raw = make([]byte, len(v))
			copy(raw, v)
		} else if message.Value != nil {
			log.DefaultLogger.Warn("Received message with unsupported Value type, skipping",
				"type", fmt.Sprintf("%T", message.Value),
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			return
		} else {
			log.DefaultLogger.Warn("Received message with no Data and no Value payload, skipping",
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			return
		}
		select {
		case msgCh <- raw:
		default:
			log.DefaultLogger.Warn("Message dropped, send buffer full", "RabbitMQ Stream", ds.Client.ToString())
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled", "RabbitMQ Stream", ds.Client.ToString())
			ds.Client.Dispose()
			return nil
		default:
		}

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
		if err != nil {
			log.DefaultLogger.Error("Failed to create consumer, stream will be re-established by Grafana", "RabbitMQ Stream", ds.Client.ToString(), "error", err)
			ds.Client.Dispose()
			return err
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled", "RabbitMQ Stream", ds.Client.ToString())
			ds.Client.Dispose()
			return nil
		case <-consumer.NotifyClose():
			log.DefaultLogger.Info(
				"RabbitMQ consumer closed, stream will be re-established by Grafana",
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			ds.Client.Dispose()
			return fmt.Errorf("RabbitMQ consumer closed unexpectedly")
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
