//go:build integration

// Integration tests against a real broker (the docker compose `rabbitmq` service):
//
//	docker compose up -d --wait rabbitmq
//	go test -tags integration ./pkg/rabbitmqclient/...
//
// RABBITMQ_HOST (default localhost) selects the broker; the management API is expected on port 15672.
package rabbitmqclient

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	streamamqp "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func brokerHost() string {
	if h := os.Getenv("RABBITMQ_HOST"); h != "" {
		return h
	}
	return "localhost"
}

func uniqueName(t *testing.T, kind string) string {
	return fmt.Sprintf("it-%s-%s-%d", kind, t.Name(), rand.Int63())
}

func testOptions(t *testing.T) (*RabbitMQStreamOptions, string, string) {
	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	opts.StreamOptions.StreamName = uniqueName(t, "stream")
	source, target := uniqueName(t, "source"), uniqueName(t, "target")
	opts.ExchangesOptions = []*ExchangeOptions{
		{Name: source, Type: "fanout", Durable: true},
		{Name: target, Type: "fanout", Durable: true},
	}
	opts.BindingsOptions = []*BindingOptions{
		{SenderName: target, ReceiverName: opts.StreamOptions.StreamName, RoutingKey: "/", IsQueueBinding: true, ShouldDisposeBinding: true},
		{SenderName: source, ReceiverName: target, RoutingKey: "/", IsQueueBinding: false, ShouldDisposeBinding: true},
	}
	t.Cleanup(func() { cleanupTopology(t, opts, source, target) })
	return opts, source, target
}

func cleanupTopology(t *testing.T, opts *RabbitMQStreamOptions, exchanges ...string) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s:%d/", opts.Host, opts.AmqpPort))
	if err != nil {
		t.Logf("cleanup: %v", err)
		return
	}
	defer closeAndLog("cleanup connection", conn)
	for _, name := range exchanges {
		if ch, err := conn.Channel(); err == nil {
			_ = ch.ExchangeDelete(name, false, false)
			closeAndLog("cleanup channel", ch)
		}
	}
	if ch, err := conn.Channel(); err == nil {
		_, _ = ch.QueueDelete(opts.StreamOptions.StreamName, false, false, false)
		closeAndLog("cleanup channel", ch)
	}
}

func connect(t *testing.T, opts *RabbitMQStreamOptions) *RabbitMQStreamClient {
	t.Helper()
	client := NewRabbitMQStreamClient().SetRabbitMQOptions(opts)
	if _, err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return client
}

func bindingsFrom(t *testing.T, exchange string) []map[string]any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://%s:15672/api/exchanges/%%2F/%s/bindings/source", brokerHost(), url.PathEscape(exchange)), nil)
	req.SetBasicAuth("guest", "guest")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("management API: %v", err)
	}
	defer closeAndLog("response body", res.Body)
	var bindings []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&bindings); err != nil {
		t.Fatalf("decoding bindings: %v", err)
	}
	return bindings
}

// Connect() runs again on every reconnect. It must not accumulate duplicate exchanges/bindings, otherwise
// disposal deletes the same object twice and aborts on the second (404) delete.
func TestConnectTwiceDoesNotDuplicateTopology(t *testing.T) {
	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer func() { _ = client.Env.Close() }()

	if _, err := client.Connect(); err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	if len(client.Exchanges) != len(opts.ExchangesOptions) || len(client.Bindings) != len(opts.BindingsOptions) {
		t.Fatalf("after two Connect() calls: %d exchanges and %d bindings, want %d and %d",
			len(client.Exchanges), len(client.Bindings), len(opts.ExchangesOptions), len(opts.BindingsOptions))
	}
}

// When the stream is deleted in RabbitMQ the consumer is closed by the broker. Disposing must then clear the
// consumer so that Reconnect() can create a new one instead of failing (and retrying) forever.
func TestDisposeStreamClearsConsumerAfterTheStreamWasDeleted(t *testing.T) {
	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer func() { _ = client.Env.Close() }()

	consumer, err := client.Consume(func(stream.ConsumerContext, *streamamqp.Message) {})
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if err := client.Env.DeleteStream(opts.StreamOptions.StreamName); err != nil {
		t.Fatalf("DeleteStream: %v", err)
	}
	select {
	case <-consumer.NotifyClose():
	case <-time.After(10 * time.Second):
		t.Fatal("the consumer was not closed after its stream was deleted")
	}

	_ = opts.StreamOptions.DisposeStream(client.Env) // may report that the consumer was already closed

	if opts.StreamOptions.Consumer != nil {
		t.Fatal("DisposeStream left the closed consumer in place; Consume() would refuse to create a new one")
	}
}

// DisposeBindings must remove exchange-to-exchange bindings, not re-declare them.
func TestDisposeBindingsRemovesExchangeToExchangeBindings(t *testing.T) {
	opts, source, _ := testOptions(t)
	client := connect(t, opts)
	defer func() { _ = client.Env.Close() }()
	if len(bindingsFrom(t, source)) != 1 {
		t.Fatalf("expected the exchange-to-exchange binding from %s to exist after Connect", source)
	}

	conn, err := client.createAmqpConnection()
	if err != nil {
		t.Fatalf("AMQP connection: %v", err)
	}
	defer closeAndLog("AMQP connection", conn)
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer closeAndLog("AMQP channel", ch)
	if _, err := client.DisposeBindings(ch); err != nil {
		t.Fatalf("DisposeBindings: %v", err)
	}

	if remaining := bindingsFrom(t, source); len(remaining) != 0 {
		t.Fatalf("exchange-to-exchange binding still exists after DisposeBindings: %v", remaining)
	}
}
