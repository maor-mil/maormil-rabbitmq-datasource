//go:build integration

package rabbitmqclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	streamamqp "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func publish(t *testing.T, exchange, routingKey, payload string) {
	t.Helper()
	if exchange == "" {
		exchange = "amq.default"
	}
	management(t, http.MethodPost, "/api/exchanges/%2F/"+url.PathEscape(exchange)+"/publish", map[string]any{
		"properties": map[string]any{}, "routing_key": routingKey, "payload": payload, "payload_encoding": "string",
	})
}

func queueMessages(t *testing.T, name string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:15672/api/queues/%%2F/%s", brokerHost(), url.PathEscape(name)), nil)
	req.SetBasicAuth("guest", "guest")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndLog("response body", res.Body)
	var queue struct {
		Messages int `json:"messages"`
	}
	_ = json.NewDecoder(res.Body).Decode(&queue)
	return queue.Messages
}

// consumeQueueInBackground runs ConsumeQueue until the test ends and collects the bodies.
func consumeQueueInBackground(t *testing.T, opts *RabbitMQStreamOptions, source Source) (<-chan string, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	bodies := make(chan string, 100)
	done := make(chan error, 1)
	client := NewRabbitMQStreamClient().SetRabbitMQOptions(opts)
	go func() { done <- client.ConsumeQueue(ctx, source, func(b []byte) { bodies <- string(b) }) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("ConsumeQueue did not stop after its context was cancelled")
		}
	})
	return bodies, done
}

// awaitBody keeps publishing (the tap queue is declared asynchronously) until marker is received.
func awaitBody(t *testing.T, bodies <-chan string, done <-chan error, publishMarker func(marker string)) {
	t.Helper()
	deadline := time.After(30 * time.Second)
	for i := 0; ; i++ {
		marker := fmt.Sprintf(`{"marker":%d}`, i)
		publishMarker(marker)
		select {
		case body := <-bodies:
			if strings.HasPrefix(body, `{"marker":`) {
				return
			}
		case err := <-done:
			t.Fatalf("ConsumeQueue stopped: %v", err)
		case <-time.After(time.Second):
		case <-deadline:
			t.Fatal("no message received within 30s")
		}
	}
}

func TestTapReceivesCopiesWithoutTakingMessagesFromApplicationQueues(t *testing.T) {
	exchange, appQueue := uniqueName(t, "topic"), uniqueName(t, "app-queue")
	management(t, http.MethodPut, "/api/exchanges/%2F/"+url.PathEscape(exchange), map[string]any{"type": "topic", "durable": false})
	t.Cleanup(func() { management(t, http.MethodDelete, "/api/exchanges/%2F/"+url.PathEscape(exchange), nil) })
	management(t, http.MethodPut, "/api/queues/%2F/"+url.PathEscape(appQueue), map[string]any{"durable": false})
	t.Cleanup(func() { management(t, http.MethodDelete, "/api/queues/%2F/"+url.PathEscape(appQueue), nil) })
	management(t, http.MethodPost, "/api/bindings/%2F/e/"+url.PathEscape(exchange)+"/q/"+url.PathEscape(appQueue), map[string]any{"routing_key": "sensors.#"})

	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	bodies, done := consumeQueueInBackground(t, opts, Source{Type: SourceQueue, QueueMode: QueueModeTap, Exchange: exchange, RoutingKey: "sensors.#"})

	awaitBody(t, bodies, done, func(m string) { publish(t, exchange, "sensors.temperature", m) })

	// Every message published also reached the application queue: the tap did not consume from it.
	// (Message counts come from the management stats, refreshed every few seconds.)
	deadline := time.Now().Add(20 * time.Second)
	for queueMessages(t, appQueue) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Second)
	}
	if n := queueMessages(t, appQueue); n == 0 {
		t.Error("the application queue lost its messages")
	}
}

func TestConsumeAnExistingQuorumQueue(t *testing.T) {
	queue := uniqueName(t, "quorum")
	management(t, http.MethodPut, "/api/queues/%2F/"+url.PathEscape(queue), map[string]any{"durable": true, "arguments": map[string]any{"x-queue-type": "quorum"}})
	t.Cleanup(func() { management(t, http.MethodDelete, "/api/queues/%2F/"+url.PathEscape(queue), nil) })

	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	bodies, done := consumeQueueInBackground(t, opts, Source{Type: SourceQueue, QueueMode: QueueModeExisting, Queue: queue})

	awaitBody(t, bodies, done, func(m string) { publish(t, "", queue, m) })
	// Consuming an existing queue is destructive: the messages are acknowledged and removed.
	deadline := time.Now().Add(15 * time.Second)
	for queueMessages(t, queue) != 0 && time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
	}
	if n := queueMessages(t, queue); n != 0 {
		t.Errorf("%d messages left in the consumed queue", n)
	}
}

func TestConsumeQueueReportsAMissingExchange(t *testing.T) {
	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	err := NewRabbitMQStreamClient().SetRabbitMQOptions(opts).
		ConsumeQueue(context.Background(), Source{Type: SourceQueue, QueueMode: QueueModeTap, Exchange: "no-such-exchange"}, func([]byte) {})
	if err == nil || !strings.Contains(err.Error(), "no-such-exchange") {
		t.Fatalf("expected an error naming the exchange, got %v", err)
	}
}

func TestConsumeAnyStreamFromAnOffset(t *testing.T) {
	name := uniqueName(t, "other-stream")
	management(t, http.MethodPut, "/api/queues/%2F/"+url.PathEscape(name), map[string]any{"durable": true, "arguments": map[string]any{"x-queue-type": "stream"}})
	t.Cleanup(func() { management(t, http.MethodDelete, "/api/queues/%2F/"+url.PathEscape(name), nil) })
	publish(t, "", name, `{"before":"consumer"}`)

	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer client.Close()
	bodies := make(chan string, 10)
	consumer, err := client.ConsumeStream(name, OffsetFirst, func(_ stream.ConsumerContext, m *streamamqp.Message) {
		body, _ := MessageBody(m.Data, m.Value)
		bodies <- string(body)
	})
	if err != nil {
		t.Fatalf("ConsumeStream: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	select {
	case body := <-bodies:
		if body != `{"before":"consumer"}` {
			t.Errorf("first message = %s", body)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("offset first did not replay the stored message")
	}
}

func TestTapOverTLSWithClientCertificate(t *testing.T) {
	const user = "e2e-tls-client"
	management(t, http.MethodPut, "/api/users/"+user, map[string]any{"password_hash": "", "tags": ""})
	management(t, http.MethodPut, "/api/permissions/%2F/"+user, map[string]any{"configure": ".*", "write": ".*", "read": ".*"})

	opts := NewRabbitMQStreamOptions()
	opts.Host, opts.IsTLS, opts.AmqpPort = brokerHost(), true, 5671
	opts.SASLMechanism, opts.User, opts.Password = SASLMechanismExternal, "", ""
	opts.TLSAuthWithCACert, opts.CACert = true, tlsFile(t, "ca.pem")
	opts.TLSAuth, opts.ClientCert, opts.ClientKey = true, tlsFile(t, "client.pem"), tlsFile(t, "client-key.pem")

	bodies, done := consumeQueueInBackground(t, opts, Source{Type: SourceQueue, QueueMode: QueueModeTap, Exchange: "amq.fanout"})
	awaitBody(t, bodies, done, func(m string) { publish(t, "amq.fanout", "", m) })
}
