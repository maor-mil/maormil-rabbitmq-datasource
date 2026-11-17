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
	"strings"
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
		{SenderName: target, ReceiverName: opts.StreamOptions.StreamName, RoutingKey: "/", IsQueueBinding: true},
		{SenderName: source, ReceiverName: target, RoutingKey: "/", IsQueueBinding: false},
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

// Connect() runs again on every reconnect. It must not accumulate duplicate exchanges/bindings.
func TestConnectTwiceDoesNotDuplicateTopology(t *testing.T) {
	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer client.Close()

	if _, err := client.Connect(); err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	if len(client.Exchanges) != len(opts.ExchangesOptions) || len(client.Bindings) != len(opts.BindingsOptions) {
		t.Fatalf("after two Connect() calls: %d exchanges and %d bindings, want %d and %d",
			len(client.Exchanges), len(client.Bindings), len(opts.ExchangesOptions), len(opts.BindingsOptions))
	}
}

func queueExists(t *testing.T, name string) bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://%s:15672/api/queues/%%2F/%s", brokerHost(), url.PathEscape(name)), nil)
	req.SetBasicAuth("guest", "guest")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("management API: %v", err)
	}
	defer closeAndLog("response body", res.Body)
	return res.StatusCode == http.StatusOK
}

// The plugin never deletes broker objects: Close only releases the consumer and the connection.
func TestCloseKeepsTheTopologyAndClearsTheConsumer(t *testing.T) {
	opts, source, target := testOptions(t)
	client := connect(t, opts)
	if _, err := client.Consume(func(stream.ConsumerContext, *streamamqp.Message) {}); err != nil {
		t.Fatalf("Consume: %v", err)
	}

	client.Close()

	if client.IsConnected() {
		t.Fatal("still connected after Close")
	}
	if opts.StreamOptions.Consumer != nil {
		t.Fatal("Close left the consumer in place; the next Consume() would refuse to create one")
	}
	if !queueExists(t, opts.StreamOptions.StreamName) {
		t.Fatal("Close deleted the stream")
	}
	if len(bindingsFrom(t, source)) != 1 || len(bindingsFrom(t, target)) != 1 {
		t.Fatal("Close removed bindings")
	}
}

// When the stream is deleted in RabbitMQ the broker closes the consumer. Closing and connecting again
// (what RunStream does before Grafana retries it) must recreate the stream and allow a new consumer.
func TestConnectRecreatesAStreamThatWasDeletedInRabbitMQ(t *testing.T) {
	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer client.Close()

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

	client.Close()
	if _, err := client.Connect(); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if !queueExists(t, opts.StreamOptions.StreamName) {
		t.Fatal("the stream was not recreated by Connect")
	}
	if _, err := client.Consume(func(stream.ConsumerContext, *streamamqp.Message) {}); err != nil {
		t.Fatalf("Consume after recreating the stream: %v", err)
	}
}

func management(t *testing.T, method, path string, body any) {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		reader = strings.NewReader(string(encoded))
	} else {
		reader = strings.NewReader("")
	}
	req, _ := http.NewRequest(method, fmt.Sprintf("http://%s:15672%s", brokerHost(), path), reader)
	req.SetBasicAuth("guest", "guest")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("management API %s %s: %v", method, path, err)
	}
	defer closeAndLog("response body", res.Body)
	if res.StatusCode >= 300 && !(method == http.MethodDelete && res.StatusCode == http.StatusNotFound) {
		t.Fatalf("management API %s %s: HTTP %d", method, path, res.StatusCode)
	}
}

// Issue #3: credentials and vhosts with URI delimiters must work for the AMQP topology connection and for
// every stream connection (locator and consumer).
func TestConnectWithSpecialCharactersInCredentialsAndVhost(t *testing.T) {
	for _, vhost := range []string{"/", "it vhost/#?"} {
		t.Run(vhost, func(t *testing.T) {
			user := fmt.Sprintf("it-user:@/#?-%d", rand.Int63())
			password := `@:/#?%&= p"ä`
			if vhost != "/" {
				management(t, http.MethodPut, "/api/vhosts/"+url.PathEscape(vhost), map[string]any{})
				t.Cleanup(func() { management(t, http.MethodDelete, "/api/vhosts/"+url.PathEscape(vhost), nil) })
			}
			management(t, http.MethodPut, "/api/users/"+url.PathEscape(user), map[string]any{"password": password, "tags": ""})
			t.Cleanup(func() { management(t, http.MethodDelete, "/api/users/"+url.PathEscape(user), nil) })
			management(t, http.MethodPut, "/api/permissions/"+url.PathEscape(vhost)+"/"+url.PathEscape(user),
				map[string]any{"configure": ".*", "write": ".*", "read": ".*"})

			opts := NewRabbitMQStreamOptions()
			opts.Host, opts.User, opts.Password, opts.VHost = brokerHost(), user, password, vhost
			opts.StreamOptions.StreamName = uniqueName(t, "stream")
			client := NewRabbitMQStreamClient().SetRabbitMQOptions(opts)
			if _, err := client.Connect(); err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer client.Close()
			if _, err := client.Consume(func(stream.ConsumerContext, *streamamqp.Message) {}); err != nil {
				t.Fatalf("Consume (separate consumer connection): %v", err)
			}
			if vhost == "/" {
				t.Cleanup(func() {
					management(t, http.MethodDelete, "/api/queues/%2F/"+url.PathEscape(opts.StreamOptions.StreamName), nil)
				})
			}
		})
	}
}
