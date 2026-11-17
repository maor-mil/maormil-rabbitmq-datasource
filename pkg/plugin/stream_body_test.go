package plugin

import (
	"testing"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
)

func TestMessageBody(t *testing.T) {
	cases := []struct {
		name    string
		message *amqp.Message
		want    string
		wantOK  bool
	}{
		{"single data section (stream protocol / AMQP 0.9.1)", &amqp.Message{Data: [][]byte{[]byte(`{"a":1}`)}}, `{"a":1}`, true},
		{"several data sections are concatenated", &amqp.Message{Data: [][]byte{[]byte(`{"a":`), []byte(`1}`)}}, `{"a":1}`, true},
		{"amqp-value binary", &amqp.Message{Value: []byte(`{"b":2}`)}, `{"b":2}`, true},
		{"amqp-value string", &amqp.Message{Value: `{"c":3}`}, `{"c":3}`, true},
		{"unsupported amqp-value", &amqp.Message{Value: int64(4)}, "", false},
		{"no body", &amqp.Message{}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := messageBody(tc.message)
			if ok != tc.wantOK || string(got) != tc.want {
				t.Fatalf("messageBody() = %q, %v; want %q, %v", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestMessageBodyReturnsACopy(t *testing.T) {
	data := []byte(`{"a":1}`)
	got, _ := messageBody(&amqp.Message{Data: [][]byte{data}})
	data[2] = 'X'
	if string(got) != `{"a":1}` {
		t.Fatalf("messageBody must copy the payload, got %q after mutating the source", got)
	}
}
