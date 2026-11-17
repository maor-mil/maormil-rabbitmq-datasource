package rabbitmqclient

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SourceType is what a panel query consumes.
type SourceType string

const (
	// SourceDatasource consumes the stream configured on the data source (the default, and the only
	// behaviour before per-query sources existed).
	SourceDatasource SourceType = "datasource"
	// SourceStream consumes any existing stream with the stream protocol (RabbitMQ 3.9+).
	SourceStream SourceType = "stream"
	// SourceQueue consumes classic or quorum queues over AMQP 1.0 (RabbitMQ 4.0+).
	SourceQueue SourceType = "queue"
)

// QueueMode selects how a SourceQueue is consumed.
type QueueMode string

const (
	// QueueModeTap binds a temporary, server-named, auto-delete queue to an exchange: the dashboard sees a copy
	// of every routed message and no application queue loses messages. The broker deletes the queue when the
	// panel stops streaming (or after a minute if it is never consumed).
	QueueModeTap QueueMode = "tap"
	// QueueModeExisting consumes an existing queue. This is destructive: consumed messages are removed from the
	// queue and competing consumers share the messages.
	QueueModeExisting QueueMode = "existing"
)

// Offset is where a SourceStream starts reading.
type Offset string

const (
	OffsetFirst Offset = "first"
	OffsetLast  Offset = "last"
	OffsetNext  Offset = "next"
)

// Source is the normalised description of what one panel query consumes. The short JSON keys keep it small,
// because it is encoded into the Grafana Live channel path.
type Source struct {
	Type       SourceType `json:"t"`
	Stream     string     `json:"s,omitempty"`
	Offset     Offset     `json:"o,omitempty"`
	QueueMode  QueueMode  `json:"m,omitempty"`
	Queue      string     `json:"q,omitempty"`
	Exchange   string     `json:"e,omitempty"`
	RoutingKey string     `json:"k,omitempty"`

	// Stream filtering (SourceStream only), see https://www.rabbitmq.com/docs/stream-filtering.
	// FilterValues selects chunks with the broker's Bloom filter (RabbitMQ 3.13+); MatchUnfiltered also
	// delivers messages published without a filter value.
	FilterValues    []string `json:"fv,omitempty"`
	MatchUnfiltered bool     `json:"mu,omitempty"`
	// SQL is an SQL filter expression evaluated by the broker for every message (RabbitMQ 4.2+). It is only
	// available over AMQP 1.0, so a stream source with SQL is consumed with the AMQP 1.0 client.
	SQL string `json:"sql,omitempty"`
}

// UsesAMQP10 reports whether the source has to be consumed over AMQP 1.0.
func (s Source) UsesAMQP10() bool {
	return s.Type == SourceQueue || (s.Type == SourceStream && s.SQL != "")
}

// Normalize trims the names, applies defaults and drops the fields that do not apply to the source type, so
// that equal sources always produce the same channel (and therefore share one consumer).
func (s Source) Normalize() Source {
	n := Source{Type: s.Type}
	switch s.Type {
	case "", SourceDatasource:
		n.Type = SourceDatasource
	case SourceStream:
		n.Stream = strings.TrimSpace(s.Stream)
		n.Offset = s.Offset
		if n.Offset == "" {
			n.Offset = OffsetFirst
		}
		n.FilterValues = normalizeFilterValues(s.FilterValues)
		n.MatchUnfiltered = s.MatchUnfiltered && len(n.FilterValues) > 0
		n.SQL = strings.TrimSpace(s.SQL)
	case SourceQueue:
		n.QueueMode = s.QueueMode
		if n.QueueMode == "" {
			n.QueueMode = QueueModeTap
		}
		if n.QueueMode == QueueModeTap {
			n.Exchange = strings.TrimSpace(s.Exchange)
			n.RoutingKey = s.RoutingKey
		} else {
			n.Queue = strings.TrimSpace(s.Queue)
		}
	}
	return n
}

// normalizeFilterValues trims, de-duplicates and sorts the values so equal filters share a channel.
func normalizeFilterValues(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// Validate reports a configuration error that the user can fix in the query editor.
func (s Source) Validate() error {
	switch s.Type {
	case SourceDatasource:
		return nil
	case SourceStream:
		if s.Stream == "" {
			return errors.New("enter the name of the stream to read")
		}
		switch s.Offset {
		case OffsetFirst, OffsetLast, OffsetNext:
			return nil
		}
		return fmt.Errorf("unsupported offset %q (use first, last or next)", s.Offset)
	case SourceQueue:
		switch s.QueueMode {
		case QueueModeTap:
			if s.Exchange == "" {
				return errors.New("enter the exchange to tap (a temporary queue is bound to it)")
			}
			return nil
		case QueueModeExisting:
			if s.Queue == "" {
				return errors.New("enter the name of the queue to consume")
			}
			return nil
		}
		return fmt.Errorf("unsupported queue mode %q (use tap or existing)", s.QueueMode)
	}
	return fmt.Errorf("unsupported source %q", s.Type)
}

// MessageBody returns a copy of an AMQP 1.0 message body: the Data section(s), or an amqp-value holding binary
// or string data. Messages published with the stream protocol or AMQP 0.9.1 carry their payload in Data.
func MessageBody(data [][]byte, value any) ([]byte, bool) {
	switch {
	case len(data) == 1:
		return append([]byte(nil), data[0]...), true
	case len(data) > 1:
		var body []byte
		for _, section := range data {
			body = append(body, section...)
		}
		return body, true
	}
	switch v := value.(type) {
	case []byte:
		return append([]byte(nil), v...), true
	case string:
		return []byte(v), true
	}
	return nil, false
}
