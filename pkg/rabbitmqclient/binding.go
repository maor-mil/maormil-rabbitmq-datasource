package rabbitmqclient

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Binding interface {
	CreateBinding(*RabbitMQStreamOptions) error
}

type BindingOptions struct {
	QueueName    string `json:"queueName"`
	RoutingKey   string `json:"routingKey"`
	ExchangeName string `json:"exchangeName"`
	NoWait       bool   `json:"noWait"`
}

func (bindingOptions *BindingOptions) CreateBinding(options *RabbitMQStreamOptions) error {
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

	err = ch.QueueBind(
		bindingOptions.QueueName,
		bindingOptions.RoutingKey,
		bindingOptions.ExchangeName,
		bindingOptions.NoWait,
		nil, // arguments - not supported at the moment
	)
	return failOnError(
		err,
		fmt.Sprintf("Failed to create the binding ((Exchange: %s) -> (Queue: %s) ; (RoutingKey: %s))",
			bindingOptions.ExchangeName,
			bindingOptions.QueueName,
			bindingOptions.RoutingKey,
		),
	)
}
