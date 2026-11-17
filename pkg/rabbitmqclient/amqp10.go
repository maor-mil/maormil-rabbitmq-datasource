package rabbitmqclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// AMQP 1.0 is spoken with github.com/Azure/go-amqp directly. (RabbitMQ's rabbitmq-amqp-go-client v1.6.0 wraps
// the same library but does not compile for 32-bit platforms such as linux/arm, which Grafana plugins ship.)

const (
	amqp10DialTimeout = 15 * time.Second
	amqp10Credit      = 256
	// tapQueueExpiry deletes a tap queue that never got (or lost) its consumer, e.g. after a Grafana crash.
	tapQueueExpiry = 60 * time.Second
)

// ErrAMQP10Unsupported is returned when the broker is older than RabbitMQ 4.0, which introduced the native
// AMQP 1.0 support (and the /queues/<name> addresses) that queue consumption relies on.
var ErrAMQP10Unsupported = errors.New("consuming queues requires RabbitMQ 4.0 or later (native AMQP 1.0)")

// amqp10Address is the go-amqp address: scheme, host and port. Credentials go through SASL and the vhost is
// selected with the "vhost:<name>" open hostname.
func amqp10Address(options *RabbitMQStreamOptions) string {
	scheme := "amqp"
	if options.IsTLS {
		scheme = "amqps"
	}
	return scheme + "://" + net.JoinHostPort(options.Host, strconv.Itoa(options.AmqpPort))
}

func amqp10SASL(options *RabbitMQStreamOptions) amqp.SASLType {
	if options.SASLMechanism == SASLMechanismExternal {
		return amqp.SASLTypeExternal("")
	}
	return amqp.SASLTypePlain(options.User, options.Password)
}

// queueAddress is the RabbitMQ 4 AMQP 1.0 address of a queue or stream ("/queues/<percent-encoded name>").
func queueAddress(name string) string {
	return "/queues/" + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
}

// dialAMQP10 opens an AMQP 1.0 connection with the data source's credentials and TLS settings and checks that
// the broker is at least major.minor (returning unsupported otherwise).
func (client *RabbitMQStreamClient) dialAMQP10(ctx context.Context, major, minor int, unsupported error) (*amqp.Conn, error) {
	options := client.RabbitMQOptions
	if err := validateSASLMechanism(options.SASLMechanism); err != nil {
		return nil, err
	}
	tlsConfig, err := buildTLSConfig(options)
	if err != nil {
		return nil, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, amqp10DialTimeout)
	defer cancel()
	conn, err := amqp.Dial(dialCtx, amqp10Address(options), &amqp.ConnOptions{
		ContainerID: "grafana-maormil-rabbitmq-datasource",
		HostName:    "vhost:" + options.VHost,
		SASLType:    amqp10SASL(options),
		TLSConfig:   tlsConfig, // its ServerName is the host (go-amqp would otherwise use HostName)
	})
	if err != nil {
		return nil, fmt.Errorf("AMQP 1.0 connection failed (%w); this requires RabbitMQ %d.%d or later", err, major, minor)
	}
	if err := requireRabbitMQ(conn.Properties(), major, minor, unsupported); err != nil {
		closeAMQP10(conn)
		return nil, err
	}
	return conn, nil
}

func closeAMQP10(conn *amqp.Conn) {
	if err := conn.Close(); err != nil {
		log.DefaultLogger.Debug("Closing the AMQP 1.0 connection failed", "error", err)
	}
}

// ConsumeQueue consumes a classic or quorum queue over AMQP 1.0 and calls handle with each message body until
// ctx is done (returns nil) or the connection fails (returns the error, so Grafana retries the stream). Each
// call uses its own connection.
func (client *RabbitMQStreamClient) ConsumeQueue(ctx context.Context, source Source, handle func(body []byte)) error {
	queue := source.Queue
	if source.QueueMode == QueueModeTap {
		tap, err := client.declareTapQueue(source)
		if err != nil {
			return err
		}
		queue = tap
	}

	conn, err := client.dialAMQP10(ctx, 4, 0, ErrAMQP10Unsupported)
	if err != nil {
		return err
	}
	defer closeAMQP10(conn)
	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		return fmt.Errorf("opening an AMQP 1.0 session failed: %w", err)
	}
	receiverOptions := &amqp.ReceiverOptions{Credit: amqp10Credit}
	tap := source.QueueMode == QueueModeTap
	if tap {
		// The tap queue only holds copies: let the broker settle on send, nothing to acknowledge.
		receiverOptions.RequestedSenderSettleMode = amqp.SenderSettleModeSettled.Ptr()
	}
	receiver, err := session.NewReceiver(ctx, queueAddress(queue), receiverOptions)
	if err != nil {
		return fmt.Errorf("consuming queue %q failed: %w", queue, err)
	}
	log.DefaultLogger.Debug("Consuming queue over AMQP 1.0", "queue", queue, "mode", source.QueueMode)

	return receive(ctx, receiver, queue, !tap, handle)
}

// declareTapQueue declares a server-named, auto-delete queue bound to the source exchange and routing key, over
// the AMQP 0.9.1 topology connection (AMQP 1.0 has no in-protocol queue declaration). The broker deletes it
// when the panel's consumer disconnects, or after tapQueueExpiry if it is never consumed.
func (client *RabbitMQStreamClient) declareTapQueue(source Source) (string, error) {
	options := client.RabbitMQOptions
	tlsConfig, err := buildTLSConfig(options)
	if err != nil {
		return "", err
	}
	conn, err := amqp091.DialConfig(amqpAddress(options), amqpConfig(options, tlsConfig))
	if err != nil {
		return "", fmt.Errorf("connecting to declare the tap queue failed: %w", err)
	}
	defer closeAndLog("AMQP connection", conn)
	ch, err := conn.Channel()
	if err != nil {
		return "", err
	}
	defer closeAndLog("AMQP channel", ch)

	queue, err := ch.QueueDeclare("", false, true, false, false, amqp091.Table{"x-expires": int32(tapQueueExpiry.Milliseconds())})
	if err != nil {
		return "", fmt.Errorf("declaring the temporary tap queue failed: %w", err)
	}
	if err := ch.QueueBind(queue.Name, source.RoutingKey, source.Exchange, false, nil); err != nil {
		return "", fmt.Errorf("binding the tap queue to exchange %q failed (does the exchange exist?): %w", source.Exchange, err)
	}
	return queue.Name, nil
}

// receive hands every message body to handle until ctx is done (nil) or the link fails (error). With accept,
// each message is accepted after it was handed over (at-least-once for consumed queues; for streams it only
// keeps the credit flowing).
func receive(ctx context.Context, receiver *amqp.Receiver, name string, accept bool, handle func([]byte)) error {
	for {
		message, err := receiver.Receive(ctx, nil)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("receiving from %q failed: %w", name, err)
		}
		if body, ok := MessageBody(message.Data, message.Value); ok {
			handle(body)
		} else {
			log.DefaultLogger.Warn("Skipping message without a supported body", "source", name, "bodyType", fmt.Sprintf("%T", message.Value))
		}
		if accept {
			if err := receiver.AcceptMessage(ctx, message); err != nil && ctx.Err() == nil {
				return fmt.Errorf("settling a message from %q failed: %w", name, err)
			}
		}
	}
}

// requireRabbitMQ checks the broker version announced in the AMQP 1.0 connection properties.
func requireRabbitMQ(properties map[string]any, major, minor int, unsupported error) error {
	version, _ := properties["version"].(string)
	if version == "" {
		return nil // not announced: let the broker reject what it does not support
	}
	parts := strings.SplitN(version, ".", 3)
	gotMajor, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	gotMinor := 0
	if len(parts) > 1 {
		gotMinor, _ = strconv.Atoi(parts[1])
	}
	if gotMajor < major || (gotMajor == major && gotMinor < minor) {
		return fmt.Errorf("%w; the broker runs RabbitMQ %s", unsupported, version)
	}
	return nil
}
