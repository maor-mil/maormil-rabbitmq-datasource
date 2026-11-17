package plugin

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

// fakeClient is a rabbitmqclient.Client whose consumer can never be created.
type fakeClient struct {
	consumeErr error
	disposed   int
}

func (c *fakeClient) IsConnected() bool                       { return true }
func (c *fakeClient) Connect() (rabbitmqclient.Client, error) { return c, nil }
func (c *fakeClient) Reconnect() rabbitmqclient.Client        { return c }
func (c *fakeClient) Dispose()                                { c.disposed++ }
func (c *fakeClient) ToString() string                        { return "fake" }
func (c *fakeClient) Consume(stream.MessagesHandler) (*stream.Consumer, error) {
	return nil, c.consumeErr
}

type discardPackets struct{}

func (discardPackets) Send(*backend.StreamPacket) error { return nil }

// runStream runs RunStream with a timeout and converts a panic into an error.
func runStream(ds *RabbitMQDatasource, timeout time.Duration) (err error, panicked bool, finished bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		err = ds.RunStream(ctx, &backend.RunStreamRequest{Path: STREAM_CHANNEL_PATH}, backend.NewStreamSender(discardPackets{}))
	}()
	select {
	case <-done:
		return err, panicked, true
	case <-time.After(timeout + time.Second):
		return nil, false, false
	}
}

func TestRunStreamReturnsAnErrorWhenTheConsumerCannotBeCreated(t *testing.T) {
	client := &fakeClient{consumeErr: errors.New("Stream Does Not Exist")}

	err, panicked, finished := runStream(NewRabbitMQDatasource(client), 2*time.Second)

	if panicked {
		t.Fatalf("RunStream panicked instead of returning the consumer error: %v", err)
	}
	if !finished {
		t.Fatal("RunStream did not return although the consumer could not be created")
	}
	if err == nil {
		t.Fatal("RunStream returned nil; Grafana would treat the stream as healthy and never retry it")
	}
}
