package rabbitmqclient

import (
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
	// Connect opens the stream connection and (re-)declares the configured topology. Declaring is
	// idempotent, so calling Connect again recreates a stream that was deleted in RabbitMQ.
	Connect() (Client, error)
	Consume(stream.MessagesHandler) (*stream.Consumer, error)
	// Close closes the consumer and the connection. It never deletes streams, exchanges or bindings.
	Close()
	ToString() string
}

type RabbitMQStreamOptions struct {
	Host                  string             `json:"host"`
	StreamPort            int                `json:"streamPort"`
	AmqpPort              int                `json:"amqpPort"`
	VHost                 string             `json:"vHost"`
	User                  string             `json:"username"`
	Password              string             `json:"password"`
	IsTLS                 bool               `json:"tlsConnection"`
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
	// Connect to the broker
	env, err := stream.NewEnvironment(
		stream.NewEnvironmentOptions().
			SetHost(client.RabbitMQOptions.Host).
			SetPort(client.RabbitMQOptions.StreamPort).
			SetVHost(client.RabbitMQOptions.VHost).
			SetUser(client.RabbitMQOptions.User).
			SetPassword(client.RabbitMQOptions.Password).
			IsTLS(client.RabbitMQOptions.IsTLS).
			//SetTLSConfig(&tls.Config{}). - TLSConfig is not supported at the moment
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
	options := client.RabbitMQOptions
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", options.User, options.Password, options.Host, options.AmqpPort, options.VHost))
	return conn, err

}

func (client *RabbitMQStreamClient) Connect() (Client, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Never leak a previous connection.
	client.close()

	log.DefaultLogger.Debug("Trying to set the RabbitMQ environment...")
	if _, err := client.SetEnv(); err != nil {
		log.DefaultLogger.Error("Couldn't set the RabbitMQ environment", "error", err)
		client.Env = nil
		return client, err
	}
	log.DefaultLogger.Debug("Successfully set the RabbitMQ environment!")

	client.SetStream()
	client.SetExchanges()
	client.SetBindings()

	// Only report "connected" once the topology is in place.
	if err := client.declareTopology(); err != nil {
		client.close()
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
