package rabbitmqclient

import "testing"

func TestRequireRabbitMQ(t *testing.T) {
	cases := []struct {
		version      string
		major, minor int
		wantErr      bool
	}{
		{"3.13.7", 4, 0, true}, {"3.9.0", 4, 0, true}, {"4.0.0", 4, 0, false}, {"4.2.1", 4, 0, false},
		{"4.1.4", 4, 2, true}, {"4.2.0", 4, 2, false}, {"5.0.1", 4, 2, false},
		{"", 4, 2, false}, {"unknown", 4, 2, false},
	}
	for _, tc := range cases {
		err := requireRabbitMQ(map[string]any{"version": tc.version}, tc.major, tc.minor, ErrAMQP10Unsupported)
		if (err != nil) != tc.wantErr {
			t.Errorf("version %q >= %d.%d: err = %v, want error %v", tc.version, tc.major, tc.minor, err, tc.wantErr)
		}
	}
}

func TestAMQP10Address(t *testing.T) {
	opts := NewRabbitMQStreamOptions()
	opts.Host = "broker"
	if got := amqp10Address(opts); got != "amqp://broker:5672" {
		t.Errorf("plain: %q", got)
	}
	opts.IsTLS, opts.AmqpPort = true, 5671
	if got := amqp10Address(opts); got != "amqps://broker:5671" {
		t.Errorf("TLS: %q", got)
	}
	if got := queueAddress("team a/#+ä"); got != "/queues/team%20a%2F%23%2B%C3%A4" {
		t.Errorf("queueAddress: %q", got)
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
