package rabbitmqclient

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Exchange interface {
	CreateExchange(*amqp.Channel) error
}

type ExchangeOptions struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Durable     bool   `json:"durable"`
	AutoDeleted bool   `json:"autoDeleted"`
	Internal    bool   `json:"internal"`
	NoWait      bool   `json:"noWait"`
}

func (exchangeOptions *ExchangeOptions) CreateExchange(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
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
