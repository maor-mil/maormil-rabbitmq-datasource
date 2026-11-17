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

func TestStreamFilterNormalization(t *testing.T) {
	s := (Source{Type: SourceStream, Stream: "s", FilterValues: []string{" b", "a", "", "b"}, MatchUnfiltered: true, SQL: "  x = 1 "}).Normalize()
	if len(s.FilterValues) != 2 || s.FilterValues[0] != "a" || s.FilterValues[1] != "b" || !s.MatchUnfiltered || s.SQL != "x = 1" {
		t.Errorf("unexpected normalisation: %+v", s)
	}
	if !s.UsesAMQP10() {
		t.Error("a stream source with SQL must use AMQP 1.0")
	}
	if s := (Source{Type: SourceStream, Stream: "s", MatchUnfiltered: true}).Normalize(); s.MatchUnfiltered || s.UsesAMQP10() {
		t.Errorf("match-unfiltered without values is meaningless and a Bloom-only filter uses the stream protocol: %+v", s)
	}
	if s := (Source{Type: SourceQueue, Exchange: "e", SQL: "x = 1", FilterValues: []string{"a"}}).Normalize(); s.SQL != "" || s.FilterValues != nil {
		t.Errorf("filters only apply to streams: %+v", s)
	}
}

func TestStreamLinkFilters(t *testing.T) {
	filters := streamLinkFilters(Source{Type: SourceStream, Stream: "s", FilterValues: []string{"emea"}, SQL: "region = 'EMEA'"}, OffsetFirst)
	if len(filters) != 4 {
		t.Fatalf("expected offset, SQL, Bloom values and match-unfiltered filters, got %d", len(filters))
	}
	if len(streamLinkFilters(Source{Type: SourceStream, Stream: "s"}, OffsetNext)) != 1 {
		t.Error("an unfiltered stream only needs the offset filter")
	}
}
