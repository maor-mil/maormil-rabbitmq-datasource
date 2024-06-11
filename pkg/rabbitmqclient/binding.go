package rabbitmqclient

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Binding interface {
	CreateBinding(*amqp.Channel) error
	DisposeBinding(*amqp.Channel) error
}

type BindingOptions struct {
	SenderName           string `json:"senderName"`
	RoutingKey           string `json:"routingKey"`
	ReceiverName         string `json:"receiverName"`
	NoWait               bool   `json:"noWait"`
	IsQueueBinding       bool   `json:"isQueueBinding"`
	ShouldDisposeBinding bool   `json:"shouldDisposeBinding"`
}

func (bindingOptions *BindingOptions) CreateBinding(ch *amqp.Channel) error {
	var err error = nil
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

func (bindingOptions *BindingOptions) DisposeBinding(ch *amqp.Channel) error {
	if !bindingOptions.ShouldDisposeBinding {
		return nil
	}
	var err error = nil
	receiverType := "Queue"
	if bindingOptions.IsQueueBinding {
		err = ch.QueueUnbind(
			bindingOptions.ReceiverName,
			bindingOptions.RoutingKey,
			bindingOptions.SenderName,
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
		fmt.Sprintf("Failed to unbind ((Exchange: %s) -> (%s: %s) ; (RoutingKey: %s))",
			bindingOptions.SenderName,
			receiverType,
			bindingOptions.ReceiverName,
			bindingOptions.RoutingKey,
		),
	)
}
