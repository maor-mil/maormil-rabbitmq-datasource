package rabbitmqclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Link filters RabbitMQ understands for stream consumers over AMQP 1.0
// (https://www.rabbitmq.com/docs/stream-filtering).
const (
	linkFilterOffset          = "rabbitmq:stream-offset-spec"
	linkFilterBloomValues     = "rabbitmq:stream-filter"
	linkFilterMatchUnfiltered = "rabbitmq:stream-match-unfiltered"
	linkFilterSQL             = "sql-filter"
	// sqlFilterDescriptor is the descriptor code of the AMQP SQL filter (amqp:sql-filter).
	sqlFilterDescriptor = 0x120

	// filterValidationTimeout bounds the check QueryData runs for SQL filters.
	filterValidationTimeout = 10 * time.Second
)

// ErrSQLFilterUnsupported is returned when the broker is older than RabbitMQ 4.2 (SQL filter expressions).
var ErrSQLFilterUnsupported = errors.New("SQL filter expressions require RabbitMQ 4.2 or later")

// openFilteredStream attaches an AMQP 1.0 receiver with the source's filters to a stream.
//
// An AMQP 1.0 broker answers the attach with only the filters it applied and silently drops the others: an
// invalid SQL expression - or a broker without SQL filters - would otherwise stream every message unfiltered.
// The filters the broker echoed back are checked, so a dropped SQL filter is reported as an error instead.
func (client *RabbitMQStreamClient) openFilteredStream(ctx context.Context, source Source, offset Offset) (*amqp.Receiver, func(), error) {
	conn, err := client.dialAMQP10(ctx, 4, 2, ErrSQLFilterUnsupported)
	if err != nil {
		return nil, nil, err
	}
	closeConn := func() { closeAMQP10(conn) }
	fail := func(err error) (*amqp.Receiver, func(), error) {
		closeConn()
		return nil, nil, err
	}

	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		return fail(fmt.Errorf("opening an AMQP 1.0 session failed: %w", err))
	}
	receiver, err := session.NewReceiver(ctx, queueAddress(source.Stream), &amqp.ReceiverOptions{
		Credit:  amqp10Credit,
		Filters: streamLinkFilters(source, offset),
	})
	if err != nil {
		return fail(fmt.Errorf("reading stream %q failed: %w", source.Stream, err))
	}
	if source.SQL != "" && receiver.LinkSourceFilterValue(linkFilterSQL) == nil {
		return fail(fmt.Errorf("RabbitMQ did not accept the SQL filter expression %q - check its syntax "+
			"(see https://www.rabbitmq.com/docs/stream-filtering#sql-filter-expressions)", source.SQL))
	}
	return receiver, closeConn, nil
}

func streamLinkFilters(source Source, offset Offset) []amqp.LinkFilter {
	filters := []amqp.LinkFilter{amqp.NewLinkFilter(linkFilterOffset, 0, string(offset))}
	if source.SQL != "" {
		filters = append(filters, amqp.NewLinkFilter(linkFilterSQL, sqlFilterDescriptor, source.SQL))
	}
	if len(source.FilterValues) > 0 {
		values := make([]any, 0, len(source.FilterValues))
		for _, v := range source.FilterValues {
			values = append(values, v)
		}
		filters = append(filters,
			amqp.NewLinkFilter(linkFilterBloomValues, 0, values),
			amqp.NewLinkFilter(linkFilterMatchUnfiltered, 0, source.MatchUnfiltered))
	}
	return filters
}

// ConsumeStreamAMQP10 consumes a stream over AMQP 1.0 with the source's SQL filter expression (plus optional
// Bloom filter values, which let the broker skip whole chunks first) until ctx is done or the link fails.
func (client *RabbitMQStreamClient) ConsumeStreamAMQP10(ctx context.Context, source Source, handle func(body []byte)) error {
	receiver, closeConn, err := client.openFilteredStream(ctx, source, source.Offset)
	if err != nil {
		return err
	}
	defer closeConn()
	log.DefaultLogger.Debug("Consuming stream over AMQP 1.0", "stream", source.Stream, "sql", source.SQL)

	return receive(ctx, receiver, source.Stream, true, handle)
}

// ValidateStreamFilter attaches (and immediately closes) a receiver with the source's filters, so that an
// invalid SQL expression, a missing stream or a too old broker is reported on the panel by QueryData instead of
// making the stream fail in the background.
func (client *RabbitMQStreamClient) ValidateStreamFilter(ctx context.Context, source Source) error {
	ctx, cancel := context.WithTimeout(ctx, filterValidationTimeout)
	defer cancel()
	_, closeConn, err := client.openFilteredStream(ctx, source, OffsetNext)
	if err != nil {
		return err
	}
	closeConn()
	return nil
}
