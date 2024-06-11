package rabbitmqclient

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	amqp "github.com/rabbitmq/amqp091-go"
	stream "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

type Client interface {
	IsConnected() bool
	Connect() (Client, error)
	Reconnect() Client
	Consume(stream.MessagesHandler) (*stream.Consumer, error)
	Dispose()
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
	TLSConfig             bool               `json:"TLSConfig"`
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
}

const timeToReconnect time.Duration = 2000 * time.Millisecond

func NewRabbitMQStreamClient() *RabbitMQStreamClient {
	return &RabbitMQStreamClient{}
}

func NewRabbitMQStreamOptions() *RabbitMQStreamOptions {
	return &RabbitMQStreamOptions{}
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
	for exchangeIndex := 0; exchangeIndex < len(client.RabbitMQOptions.ExchangesOptions); exchangeIndex += 1 {
		client.Exchanges = append(client.Exchanges, client.RabbitMQOptions.ExchangesOptions[exchangeIndex])
	}
	return client
}

func (client *RabbitMQStreamClient) SetBindings() *RabbitMQStreamClient {
	for bindingIndex := 0; bindingIndex < len(client.RabbitMQOptions.BindingsOptions); bindingIndex += 1 {
		client.Bindings = append(client.Bindings, client.RabbitMQOptions.BindingsOptions[bindingIndex])
	}
	return client
}

func (client *RabbitMQStreamClient) CreateStream() (*RabbitMQStreamClient, error) {
	return client, client.Stream.CreateStream(client.Env)
}

func (client *RabbitMQStreamClient) CreateExchanges(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for exchangeIndex := 0; exchangeIndex < len(client.Exchanges); exchangeIndex += 1 {
		if err := client.Exchanges[exchangeIndex].CreateExchange(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) DisposeExchanges(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for exchangeIndex := 0; exchangeIndex < len(client.Exchanges); exchangeIndex += 1 {
		if err := client.Exchanges[exchangeIndex].DisposeExchange(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) CreateBindings(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for bindingIndex := 0; bindingIndex < len(client.Bindings); bindingIndex += 1 {
		if err := client.Bindings[bindingIndex].CreateBinding(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) DisposeBindings(ch *amqp.Channel) (*RabbitMQStreamClient, error) {
	for bindingIndex := 0; bindingIndex < len(client.Bindings); bindingIndex += 1 {
		if err := client.Bindings[bindingIndex].DisposeBinding(ch); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (client *RabbitMQStreamClient) IsConnected() bool {
	return !client.Env.IsClosed()
}

func (client *RabbitMQStreamClient) createAmqpConnection() (*amqp.Connection, error) {
	options := client.RabbitMQOptions
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", options.User, options.Password, options.Host, options.AmqpPort, options.VHost))
	return conn, err

}

func (client *RabbitMQStreamClient) Connect() (Client, error) {
	log.DefaultLogger.Debug("Trying to set the RabbitMQ environment...")
	_, err := client.SetEnv()
	if err != nil {
		log.DefaultLogger.Error("Couldn't set the RabbitMQ environment", "error", err)
		return client, err
	}
	log.DefaultLogger.Debug("Successfully set the RabbitMQ environment!")

	log.DefaultLogger.Debug("Trying to set the RabbitMQ objects...")
	client.SetStream()
	client.SetExchanges()
	client.SetBindings()
	log.DefaultLogger.Debug("Successfully set the RabbitMQ objects!")

	log.DefaultLogger.Debug("Create new channel to the RabbitMQ...")
	conn, err := client.createAmqpConnection()
	if err != nil {
		log.DefaultLogger.Error("Couldn't connect to the RabbitMQ with AMQP connection", "error", err)
		return client, err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return client, err
	}
	defer ch.Close()
	log.DefaultLogger.Debug("Successfully created new channel to the RabbitMQ...")

	log.DefaultLogger.Debug("Trying to create the RabbitMQ objects...")
	_, err = client.CreateExchanges(ch)
	if err != nil {
		return client, err
	}
	_, err = client.CreateStream()
	if err != nil {
		return client, err
	}
	_, err = client.CreateBindings(ch)
	if err != nil {
		return client, err
	}
	log.DefaultLogger.Debug("Successfully created the RabbitMQ objects!")

	return client, nil
}

func (client *RabbitMQStreamClient) CloseConnection() error {
	if err := client.Stream.DisposeStream(client.Env); err != nil {
		return err
	} else {
		log.DefaultLogger.Debug("Disposed Stream", "RabbitMQ Stream", client.ToString())
	}

	log.DefaultLogger.Debug("Create new channel to the RabbitMQ...")
	conn, err := client.createAmqpConnection()
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	log.DefaultLogger.Debug("Successfully created new channel to the RabbitMQ...")

	if _, err := client.DisposeBindings(ch); err != nil {
		return err
	} else {
		log.DefaultLogger.Debug("Disposed bindings", "RabbitMQ Stream", client.ToString())
	}

	if _, err := client.DisposeExchanges(ch); err != nil {
		return err
	} else {
		log.DefaultLogger.Debug("Disposed exchanges", "RabbitMQ Stream", client.ToString())
	}

	if err := client.Env.Close(); err != nil {
		return err
	} else {
		log.DefaultLogger.Debug("Closed RabbitMQ environment", "RabbitMQ Stream", client.ToString())
	}

	return nil
}

func (client *RabbitMQStreamClient) Reconnect() Client {
	for {
		time.Sleep(timeToReconnect)
		log.DefaultLogger.Debug("Trying to reconnect to RabbitMQ", "RabbitMQ Stream", client.ToString())

		err := client.CloseConnection()
		if err != nil {
			continue
		}
		_, err = client.Connect()
		if err != nil {
			continue
		}
		break
	}
	return client
}

func (client *RabbitMQStreamClient) Consume(messageHandler stream.MessagesHandler) (*stream.Consumer, error) {
	return client.Stream.Consume(client.Env, messageHandler)
}

func (client *RabbitMQStreamClient) Dispose() {
	if client.IsConnected() {
		log.DefaultLogger.Debug("Disposing RabbitMQ Stream", "RabbitMQ Stream", client.ToString())
		err := client.CloseConnection()
		if err != nil {
			log.DefaultLogger.Error("Error disposing RabbitMQ Stream", "error", err)
		}
	}
}

func (client *RabbitMQStreamClient) ToString() string {
	return fmt.Sprintf(
		"{ Host: %s, VHost: %s, StreamName: %v }",
		client.RabbitMQOptions.Host,
		client.RabbitMQOptions.VHost,
		client.RabbitMQOptions.StreamOptions.StreamName,
	)
}
