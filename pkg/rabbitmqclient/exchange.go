package rabbitmqclient

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Exchange interface {
	CreateExchange(*RabbitMQStreamOptions) error
	DisposeExchange(*RabbitMQStreamOptions) error
}

type ExchangeOptions struct {
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	Durable               bool   `json:"durable"`
	AutoDeleted           bool   `json:"autoDeleted"`
	Internal              bool   `json:"internal"`
	NoWait                bool   `json:"noWait"`
	ShouldDisposeExchange bool   `json:"shouldDisposeExchange"`
	DisposeIfUnused       bool   `json:"disposeIfUnused"`
}

func (exchangeOptions *ExchangeOptions) CreateExchange(options *RabbitMQStreamOptions) error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", options.User, options.Password, options.Host, options.AmqpPort, options.VHost))
	if err != nil {
		return failOnError(err, fmt.Sprintf("Failed to connect to RabbitMQ: %s", options.Host))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return failOnError(err, fmt.Sprintf("Failed to open a channel in RabbitMQ: %s", options.Host))
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		exchangeOptions.Name,
		exchangeOptions.Type,
		exchangeOptions.Durable,
		exchangeOptions.AutoDeleted,
		exchangeOptions.Internal,
		exchangeOptions.NoWait,
		nil, // arguments - not supported at the moment
	)
	return failOnError(err, fmt.Sprintf("Failed to create the exchange %s", exchangeOptions.Name))
}

func (exchangeOptions *ExchangeOptions) DisposeExchange(options *RabbitMQStreamOptions) error {
	if !exchangeOptions.ShouldDisposeExchange {
		return nil
	}

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", options.User, options.Password, options.Host, options.AmqpPort, options.VHost))
	if err != nil {
		return failOnError(err, fmt.Sprintf("Failed to connect to RabbitMQ: %s", options.Host))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return failOnError(err, fmt.Sprintf("Failed to open a channel in RabbitMQ: %s", options.Host))
	}
	defer ch.Close()

	err = ch.ExchangeDelete(
		exchangeOptions.Name,
		exchangeOptions.DisposeIfUnused,
		exchangeOptions.NoWait,
	)
	return failOnError(err, fmt.Sprintf("Failed to delete the exchange %s", exchangeOptions.Name))
}
