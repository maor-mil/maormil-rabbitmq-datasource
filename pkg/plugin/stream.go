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

	// Cancelled when RunStream returns (for any reason), so the sender goroutine below never outlives it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	framer := NewFramer()
	msgCh := make(chan []byte, 256)

	// Sender goroutine: decouples frame building and SendFrame from the RabbitMQ
	// consumer callback, which runs on the stream client's own goroutine.
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
		raw, ok := messageBody(message)
		if !ok {
			log.DefaultLogger.Warn("Skipping message without a supported body",
				"bodyType", fmt.Sprintf("%T", message.Value),
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			return
		}
		// Block (instead of dropping the message) while the sender is busy: the stream client only
		// requests more messages from RabbitMQ once this callback returns, which gives natural
		// back-pressure. The select still returns as soon as the stream is stopped.
		select {
		case msgCh <- raw:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled", "RabbitMQ Stream", ds.Client.ToString())
			ds.Client.Close()
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
			ds.Client.Close()
			return err
		}

		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Stopped streaming - Context Canceled", "RabbitMQ Stream", ds.Client.ToString())
			ds.Client.Close()
			return nil
		case <-consumer.NotifyClose():
			log.DefaultLogger.Info(
				"RabbitMQ consumer closed, stream will be re-established by Grafana",
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			ds.Client.Close()
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

// messageBody returns a copy of the message body.
//
// Messages published with the stream protocol or AMQP 0.9.1 (including through the management UI)
// carry their payload in AMQP 1.0 Data section(s); AMQP 1.0 publishers may instead use an
// amqp-value section holding binary or string data.
func messageBody(message *amqp.Message) ([]byte, bool) {
	switch {
	case len(message.Data) == 1:
		return append([]byte(nil), message.Data[0]...), true
	case len(message.Data) > 1:
		var body []byte
		for _, section := range message.Data {
			body = append(body, section...)
		}
		return body, true
	}
	switch v := message.Value.(type) {
	case []byte:
		return append([]byte(nil), v...), true
	case string:
		return []byte(v), true
	}
	return nil, false
}
