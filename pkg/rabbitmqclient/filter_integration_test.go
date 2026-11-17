//go:build integration

package rabbitmqclient

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	streamamqp "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

// filteredStream declares a stream and publishes one message per region: the AMQP 0.9.1 header
// x-stream-filter-value feeds the Bloom filter, and the plain "region" header becomes an application property
// that SQL filters can use.
func filteredStream(t *testing.T) string {
	t.Helper()
	name := uniqueName(t, "filtered")
	management(t, http.MethodPut, "/api/queues/%2F/"+url.PathEscape(name),
		map[string]any{"durable": true, "arguments": map[string]any{"x-queue-type": "stream"}})
	t.Cleanup(func() { management(t, http.MethodDelete, "/api/queues/%2F/"+url.PathEscape(name), nil) })
	for _, region := range []string{"emea", "apac", "amer"} {
		management(t, http.MethodPost, "/api/exchanges/%2F/amq.default/publish", map[string]any{
			"properties":       map[string]any{"headers": map[string]any{"x-stream-filter-value": region, "region": strings.ToUpper(region)}},
			"routing_key":      name,
			"payload":          `{"region":"` + region + `"}`,
			"payload_encoding": "string",
		})
	}
	return name
}

// collect gathers bodies for a few seconds (enough for the stored messages to be replayed).
func collect(bodies <-chan string) []string {
	var got []string
	timeout := time.After(5 * time.Second)
	for {
		select {
		case b := <-bodies:
			got = append(got, b)
		case <-timeout:
			sort.Strings(got)
			return got
		}
	}
}

func TestBloomFilterOverTheStreamProtocol(t *testing.T) {
	name := filteredStream(t)
	opts, _, _ := testOptions(t)
	client := connect(t, opts)
	defer client.Close()

	bodies := make(chan string, 10)
	consumer, err := client.ConsumeStream(Source{Type: SourceStream, Stream: name, Offset: OffsetFirst, FilterValues: []string{"emea", "amer"}},
		func(_ stream.ConsumerContext, m *streamamqp.Message) {
			body, _ := MessageBody(m.Data, m.Value)
			bodies <- string(body)
		})
	if err != nil {
		t.Fatalf("ConsumeStream: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	if got := strings.Join(collect(bodies), ","); got != `{"region":"amer"},{"region":"emea"}` {
		t.Errorf("Bloom filter delivered %s", got)
	}
}

func TestSQLFilterOverAMQP10(t *testing.T) {
	name := filteredStream(t)
	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	client := NewRabbitMQStreamClient().SetRabbitMQOptions(opts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bodies := make(chan string, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.ConsumeStreamAMQP10(ctx, Source{Type: SourceStream, Stream: name, Offset: OffsetFirst, SQL: "region IN ('APAC', 'AMER')"},
			func(b []byte) { bodies <- string(b) })
	}()

	if got := strings.Join(collect(bodies), ","); got != `{"region":"amer"},{"region":"apac"}` {
		t.Errorf("SQL filter delivered %s", got)
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("ConsumeStreamAMQP10: %v", err)
	}
}

func TestValidateStreamFilter(t *testing.T) {
	name := filteredStream(t)
	opts := NewRabbitMQStreamOptions()
	opts.Host = brokerHost()
	client := NewRabbitMQStreamClient().SetRabbitMQOptions(opts)

	if err := client.ValidateStreamFilter(context.Background(), Source{Type: SourceStream, Stream: name, SQL: "region = 'EMEA'"}); err != nil {
		t.Errorf("a valid SQL filter was rejected: %v", err)
	}
	if err := client.ValidateStreamFilter(context.Background(), Source{Type: SourceStream, Stream: name, SQL: "region ==== 'EMEA'"}); err == nil {
		t.Error("an invalid SQL filter was accepted")
	}
	if err := client.ValidateStreamFilter(context.Background(), Source{Type: SourceStream, Stream: "no-such-stream", SQL: "x = 1"}); err == nil {
		t.Error("a missing stream was accepted")
	}
}
