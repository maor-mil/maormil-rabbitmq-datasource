package rabbitmqclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	amqp "github.com/rabbitmq/amqp091-go"
	stream "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

// Client is the RabbitMQ connection of one data source instance.
//
// The plugin declares the configured topology (exchanges, the stream, bindings) but never deletes
// anything in RabbitMQ: Close only releases the consumer and the connections.
type Client interface {
	IsConnected() bool
	// Connect opens the stream connection if needed and (re-)declares the configured topology. Declaring is
	// idempotent, so calling Connect again recreates a stream that was deleted in RabbitMQ.
	Connect() (Client, error)
	// Consume starts the (single) consumer of the data source's own stream.
	Consume(stream.MessagesHandler) (*stream.Consumer, error)
	// CloseConsumer closes the consumer of the data source's own stream, keeping the connection for others.
	CloseConsumer()
	// ConsumeStream starts an additional consumer on any existing stream; the caller closes it.
	ConsumeStream(name string, offset Offset, handler stream.MessagesHandler) (*stream.Consumer, error)
	// ConsumeQueue consumes a classic/quorum queue over AMQP 1.0 until ctx is done.
	ConsumeQueue(ctx context.Context, source Source, handle func(body []byte)) error
	// Close closes every consumer and the connection. It never deletes streams, exchanges or bindings.
	Close()
	ToString() string
}

type RabbitMQStreamOptions struct {
	Host       string `json:"host"`
	StreamPort int    `json:"streamPort"`
	AmqpPort   int    `json:"amqpPort"`
	VHost      string `json:"vHost"`
	User       string `json:"username"`
	Password   string `json:"password"`
	IsTLS      bool   `json:"tlsConnection"`
	// SASLMechanism is "PLAIN" (username/password, default) or "EXTERNAL" (TLS client certificate).
	SASLMechanism string `json:"saslMechanism"`
	TLSOptions
	RequestedHeartbeat    time.Duration      `json:"requestedHeartbeat"`
	RequestedMaxFrameSize int                `json:"requestedMaxFrameSize"`
	WriteBuffer           int                `json:"writeBuffer"`
	ReadBuffer            int                `json:"readBuffer"`
	NoDelay               bool               `json:"noDelay"`
	StreamOptions         *StreamOptions     `json:"streamOptions"`
	ExchangesOptions      []*ExchangeOptions `json:"exchangesOptions"`
	BindingsOptions       []*BindingOptions  `json:"bindingsOptions"`
}

type RabbitMQStreamClient struct {
	RabbitMQOptions *RabbitMQStreamOptions
	Env             *stream.Environment
	Stream          Stream
	Exchanges       []Exchange
	Bindings        []Binding

	// tlsConfig is built from the options on Connect; nil when TLS is disabled.
	tlsConfig *tls.Config

	// mu serialises Connect, Consume, Close and IsConnected, which are called concurrently by
	// RunStream, CheckHealth and instance disposal.
	mu sync.Mutex
}

func NewRabbitMQStreamClient() *RabbitMQStreamClient {
	return &RabbitMQStreamClient{}
}

func (client *RabbitMQStreamClient) SetRabbitMQOptions(rabbitmqStreamOptions *RabbitMQStreamOptions) *RabbitMQStreamClient {
	client.RabbitMQOptions = rabbitmqStreamOptions
	return client
}

func (client *RabbitMQStreamClient) SetEnv() (*RabbitMQStreamClient, error) {
	user, password, vhost := streamCredentials(client.RabbitMQOptions)
	options := stream.NewEnvironmentOptions().
		SetHost(client.RabbitMQOptions.Host).
		SetPort(client.RabbitMQOptions.StreamPort).
		SetVHost(vhost).
		SetUser(user).
		SetPassword(password).
		IsTLS(client.RabbitMQOptions.IsTLS)
	if client.tlsConfig != nil {
		options = options.SetTLSConfig(client.tlsConfig)
	}
	if client.RabbitMQOptions.SASLMechanism == SASLMechanismExternal {
		options = options.SetSaslConfiguration(stream.SaslConfigurationExternal)
	}
	// Connect to the broker
	env, err := stream.NewEnvironment(
		options.
			SetRequestedHeartbeat(client.RabbitMQOptions.RequestedHeartbeat * time.Second).
			SetRequestedMaxFrameSize(client.RabbitMQOptions.RequestedMaxFrameSize).
			SetWriteBuffer(client.RabbitMQOptions.WriteBuffer).
			SetReadBuffer(client.RabbitMQOptions.ReadBuffer).
			SetNoDelay(client.RabbitMQOptions.NoDelay).
			SetAddressResolver(stream.AddressResolver{
				Host: client.RabbitMQOptions.Host,
				Port: client.RabbitMQOptions.StreamPort,
			},
			),
	)

	client.Env = env

	return client, err
}

func (client *RabbitMQStreamClient) SetStream() *RabbitMQStreamClient {
	client.Stream = client.RabbitMQOptions.StreamOptions
	return client
}

func (client *RabbitMQStreamClient) SetExchanges() *RabbitMQStreamClient {
	client.Exchanges = nil
	for _, exchange := range client.RabbitMQOptions.ExchangesOptions {
		client.Exchanges = append(client.Exchanges, exchange)
	}
	return client
}

func (client *RabbitMQStreamClient) SetBindings() *RabbitMQStreamClient {
	client.Bindings = nil
	for _, binding := range client.RabbitMQOptions.BindingsOptions {
		client.Bindings = append(client.Bindings, binding)
	}
	return client
}

func (client *RabbitMQStreamClient) CreateStream() (*RabbitMQStreamClient, error) {
	return client, client.Stream.CreateStream(client.Env)
}

func (client *RabbitMQStreamClient) CreateExchanges(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for _, exchange := range client.Exchanges {
		if err := exchange.CreateExchange(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) CreateBindings(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for _, binding := range client.Bindings {
		if err := binding.CreateBinding(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) IsConnected() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.isConnected()
}

func (client *RabbitMQStreamClient) isConnected() bool {
	return client.Env != nil && !client.Env.IsClosed()
}

// closeAndLog closes c and logs the error instead of silently discarding it.
func closeAndLog(what string, c interface{ Close() error }) {
	if err := c.Close(); err != nil {
		log.DefaultLogger.Debug("Failed to close "+what, "error", err)
	}
}

func (client *RabbitMQStreamClient) createAmqpConnection() (*amqp.Connection, error) {
	return amqp.DialConfig(amqpAddress(client.RabbitMQOptions), amqpConfig(client.RabbitMQOptions, client.tlsConfig))
}

func (client *RabbitMQStreamClient) Connect() (Client, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if !client.isConnected() {
		// Drop leftovers of a broken connection before opening a new one.
		client.close()

		if err := validateSASLMechanism(client.RabbitMQOptions.SASLMechanism); err != nil {
			return client, err
		}
		tlsConfig, err := buildTLSConfig(client.RabbitMQOptions)
		if err != nil {
			return client, err
		}
		client.tlsConfig = tlsConfig

		log.DefaultLogger.Debug("Trying to set the RabbitMQ environment...")
		if _, err := client.SetEnv(); err != nil {
			log.DefaultLogger.Error("Couldn't set the RabbitMQ environment", "error", err)
			client.Env = nil
			return client, err
		}
		log.DefaultLogger.Debug("Successfully set the RabbitMQ environment!")
	}

	client.SetStream()
	client.SetExchanges()
	client.SetBindings()

	if err := client.declareTopology(); err != nil {
		return client, err
	}
	log.DefaultLogger.Debug("Successfully declared the RabbitMQ objects!")

	return client, nil
}

// declareTopology declares the exchanges, the stream and the bindings (in that order, so bindings can
// reference both). Every declaration is idempotent.
func (client *RabbitMQStreamClient) declareTopology() error {
	conn, err := client.createAmqpConnection()
	if err != nil {
		log.DefaultLogger.Error("Couldn't connect to the RabbitMQ with AMQP connection", "error", err)
		return err
	}
	defer closeAndLog("AMQP connection", conn)
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer closeAndLog("AMQP channel", ch)

	if _, err := client.CreateExchanges(ch); err != nil {
		return err
	}
	if _, err := client.CreateStream(); err != nil {
		return err
	}
	if _, err := client.CreateBindings(ch); err != nil {
		return err
	}
	return nil
}

func (client *RabbitMQStreamClient) Consume(messageHandler stream.MessagesHandler) (*stream.Consumer, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if !client.isConnected() {
		return nil, fmt.Errorf("not connected to RabbitMQ")
	}
	return client.Stream.Consume(client.Env, messageHandler)
}

func (client *RabbitMQStreamClient) CloseConsumer() {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.Stream != nil {
		if err := client.Stream.CloseConsumer(); err != nil {
			log.DefaultLogger.Debug("Closing the consumer failed", "RabbitMQ Stream", client.ToString(), "error", err)
		}
	}
}

func (client *RabbitMQStreamClient) ConsumeStream(name string, offset Offset, handler stream.MessagesHandler) (*stream.Consumer, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if !client.isConnected() {
		return nil, fmt.Errorf("not connected to RabbitMQ")
	}
	// Anonymous (unnamed) consumer: several panels may read the same stream, and no offset is stored.
	return client.Env.NewConsumer(name, handler, stream.NewConsumerOptions().
		SetOffset(offsetSpecification(offset)).
		SetCRCCheck(client.RabbitMQOptions.StreamOptions.Crc))
}

func offsetSpecification(offset Offset) stream.OffsetSpecification {
	switch offset {
	case OffsetLast:
		return stream.OffsetSpecification{}.Last()
	case OffsetNext:
		return stream.OffsetSpecification{}.Next()
	default:
		return stream.OffsetSpecification{}.First()
	}
}

func (client *RabbitMQStreamClient) Close() {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.close()
}

// close releases the consumer and the stream environment. Errors are logged, not returned: closing is
// best-effort, so a broken connection never prevents the next Connect.
func (client *RabbitMQStreamClient) close() {
	if client.Stream != nil {
		if err := client.Stream.CloseConsumer(); err != nil {
			log.DefaultLogger.Debug("Closing the consumer failed", "RabbitMQ Stream", client.ToString(), "error", err)
		}
	}
	if client.isConnected() {
		if err := client.Env.Close(); err != nil {
			log.DefaultLogger.Debug("Closing the RabbitMQ environment failed", "RabbitMQ Stream", client.ToString(), "error", err)
		} else {
			log.DefaultLogger.Debug("Closed RabbitMQ environment", "RabbitMQ Stream", client.ToString())
		}
	}
	client.Env = nil
}

func (client *RabbitMQStreamClient) ToString() string {
	return fmt.Sprintf(
		"{ Host: %s, VHost: %s, StreamName: %v }",
		client.RabbitMQOptions.Host,
		client.RabbitMQOptions.VHost,
		client.RabbitMQOptions.StreamOptions.StreamName,
	)
}
