package rabbitmqclient

import "testing"

func TestRequireRabbitMQ4(t *testing.T) {
	for version, wantErr := range map[string]bool{"3.13.7": true, "3.9.0": true, "4.0.0": false, "4.2.1": false, "": false, "unknown": false} {
		err := requireRabbitMQ4(map[string]any{"version": version})
		if (err != nil) != wantErr {
			t.Errorf("version %q: err = %v, want error %v", version, err, wantErr)
		}
	}
}

func TestAMQP10Address(t *testing.T) {
	opts := NewRabbitMQStreamOptions()
	opts.Host = "broker"
	if got := amqp10Address(opts); got != "amqp://broker:5672/%2F" {
		t.Errorf("default vhost: %q", got)
	}
	opts.IsTLS, opts.AmqpPort, opts.VHost = true, 5671, "team a/#"
	if got := amqp10Address(opts); got != "amqps://broker:5671/team%20a%2F%23" {
		t.Errorf("TLS + special vhost: %q", got)
	}
}

func TestSourceDefaults(t *testing.T) {
	if s := (Source{Type: SourceStream, Stream: " s "}).Normalize(); s.Offset != OffsetFirst || s.Stream != "s" {
		t.Errorf("stream defaults: %+v", s)
	}
	if s := (Source{Type: SourceQueue, Exchange: "e"}).Normalize(); s.QueueMode != QueueModeTap {
		t.Errorf("queue mode should default to tap: %+v", s)
	}
	if s := (Source{}).Normalize(); s.Type != SourceDatasource {
		t.Errorf("empty source should be the data source stream: %+v", s)
	}
}
