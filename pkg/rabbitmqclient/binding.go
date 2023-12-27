package rabbitmqclient

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Binding interface {
	CreateBinding(*RabbitMQStreamOptions) error
}

type BindingOptions struct {
	SenderName     string `json:"senderName"`
	RoutingKey     string `json:"routingKey"`
	ReceiverName   string `json:"receiverName"`
	NoWait         bool   `json:"noWait"`
	IsQueueBinding bool   `json:"isQueueBinding"`
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

	receiverType := "Queue"
	if bindingOptions.IsQueueBinding {
		err = ch.QueueBind(
			bindingOptions.ReceiverName,
			bindingOptions.RoutingKey,
			bindingOptions.SenderName,
			bindingOptions.NoWait,
			nil, // arguments - not supported at the moment
		)
	} else {
		err = ch.ExchangeBind(
			bindingOptions.ReceiverName,
			bindingOptions.RoutingKey,
			bindingOptions.SenderName,
			bindingOptions.NoWait,
			nil, // arguments - not supported at the moment
		)
		receiverType = "Exchange"
	}
	return failOnError(
		err,
		fmt.Sprintf("Failed to create the binding ((Exchange: %s) -> (%s: %s) ; (RoutingKey: %s))",
			bindingOptions.SenderName,
			receiverType,
			bindingOptions.ReceiverName,
			bindingOptions.RoutingKey,
		),
	)
}
