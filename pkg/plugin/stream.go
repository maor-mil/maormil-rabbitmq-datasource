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

// RunStream streams one Live channel. Grafana runs it once per channel (shared by all panels with the same
// source) while the channel has subscribers, cancels ctx when the last one leaves, and restarts it with a
// backoff when it returns an error - that is how the plugin reconnects.
func (ds *RabbitMQDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	source, err := ds.sourceFromPath(req.Path)
	if err != nil {
		return err
	}
	log.DefaultLogger.Info("Called RunStream method", "RabbitMQ Stream", ds.Client.ToString(), "source", source)

	// Cancelled when RunStream returns (for any reason), so the sender goroutine never outlives it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	push := ds.startSender(ctx, sender)

	switch source.Type {
	case rabbitmqclient.SourceStream:
		return ds.runStreamSource(ctx, source, push)
	case rabbitmqclient.SourceQueue:
		return ds.Client.ConsumeQueue(ctx, source, push)
	default:
		return ds.runDatasourceStream(ctx, push)
	}
}

// startSender turns message bodies into frames and sends them, on its own goroutine, so a slow SendFrame never
// runs on (or panics in) the RabbitMQ client's goroutine. The returned push blocks while the sender is busy
// (back-pressure instead of dropping messages) and returns as soon as ctx is done.
func (ds *RabbitMQDatasource) startSender(ctx context.Context, sender *backend.StreamSender) func([]byte) {
	framer := NewFramer()
	msgCh := make(chan []byte, 256)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-msgCh:
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
					frame, err := framer.ToFrame(NewTimestampedMessage(raw))
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

	return func(raw []byte) {
		select {
		case msgCh <- raw:
		case <-ctx.Done():
		}
	}
}

// streamHandler adapts push to the stream client's callback.
func (ds *RabbitMQDatasource) streamHandler(push func([]byte)) stream.MessagesHandler {
	return func(_ stream.ConsumerContext, message *amqp.Message) {
		raw, ok := messageBody(message)
		if !ok {
			log.DefaultLogger.Warn("Skipping message without a supported body",
				"bodyType", fmt.Sprintf("%T", message.Value),
				"RabbitMQ Stream", ds.Client.ToString(),
			)
			return
		}
		push(raw)
	}
}

// runDatasourceStream consumes the stream configured on the data source (declaring its topology first).
func (ds *RabbitMQDatasource) runDatasourceStream(ctx context.Context, push func([]byte)) error {
	if _, err := ds.Client.Connect(); err != nil {
		return err
	}
	consumer, err := ds.Client.Consume(ds.streamHandler(push))
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
		// Keep the connection: other panels of this data source may still be streaming.
		ds.Client.CloseConsumer()
		return nil
	case <-consumer.NotifyClose():
		log.DefaultLogger.Info("RabbitMQ consumer closed, stream will be re-established by Grafana", "RabbitMQ Stream", ds.Client.ToString())
		// The connection may be broken: start over (the other streams of this data source restart too).
		ds.Client.Close()
		return fmt.Errorf("RabbitMQ consumer closed unexpectedly")
	}
}

// runStreamSource consumes any existing stream chosen in the query editor.
func (ds *RabbitMQDatasource) runStreamSource(ctx context.Context, source rabbitmqclient.Source, push func([]byte)) error {
	if _, err := ds.Client.Connect(); err != nil {
		return err
	}
	consumer, err := ds.Client.ConsumeStream(source.Stream, source.Offset, ds.streamHandler(push))
	if err != nil {
		return fmt.Errorf("reading stream %q failed: %w", source.Stream, err)
	}

	select {
	case <-ctx.Done():
		if err := consumer.Close(); err != nil {
			log.DefaultLogger.Debug("Closing the stream consumer failed", "stream", source.Stream, "error", err)
		}
		return nil
	case <-consumer.NotifyClose():
		ds.Client.Close()
		return fmt.Errorf("the consumer of stream %q closed unexpectedly", source.Stream)
	}
}

// SubscribeStream accepts subscriptions to valid channels only.
func (ds *RabbitMQDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	if _, err := ds.sourceFromPath(req.Path); err != nil {
		log.DefaultLogger.Warn("Rejected subscription to an invalid channel", "path", req.Path, "error", err)
		return &backend.SubscribeStreamResponse{Status: backend.SubscribeStreamStatusNotFound}, nil
	}
	return &backend.SubscribeStreamResponse{Status: backend.SubscribeStreamStatusOK}, nil
}

// PublishStream just returns permission denied in this case, since in this example we don't want the user to send stream data.
// Permissions verifications could be done here. Check backend.StreamHandler docs for more details.
func (ds *RabbitMQDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	log.DefaultLogger.Info("Called PublishStream method", "RabbitMQ Stream", ds.Client.ToString())
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}

// messageBody returns a copy of the body of a message received with the stream protocol.
func messageBody(message *amqp.Message) ([]byte, bool) {
	return rabbitmqclient.MessageBody(message.Data, message.Value)
}
