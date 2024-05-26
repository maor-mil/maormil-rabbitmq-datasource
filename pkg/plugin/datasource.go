package plugin

import (
	"context"
	"encoding/json"

	"github.com/maor2475/rabbitmq-datasource/pkg/rabbitmqclient"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces- only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*RabbitMQDatasource)(nil)
	_ backend.CheckHealthHandler    = (*RabbitMQDatasource)(nil)
	_ backend.StreamHandler         = (*RabbitMQDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*RabbitMQDatasource)(nil)
)

func NewRabbitMQInstance(ctx context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Debug("Creating new RabbitMQ Instance...")

	client, err := getDatasourceSettings(s)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Debug("New RabbitMQ Instance Datasource settings were set!")

	_, err = client.Connect()
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Debug("Successfully connected to the RabbitMQ!")

	return NewRabbitMQDatasource(client), nil
}

type RabbitMQDatasource struct {
	Client rabbitmqclient.Client
}

func NewRabbitMQDatasource(client rabbitmqclient.Client) *RabbitMQDatasource {
	return &RabbitMQDatasource{
		Client: client,
	}
}

// Dispose here tells plugin SDK that plugin wants to clean up resources
// when a new instance created. As soon as datasource settings change detected
// by SDK old datasource instance will be disposed and a new one will be created
// using RabbitMQDatasource factory function.
func (ds *RabbitMQDatasource) Dispose() {
	ds.Client.Dispose()
}

func getDatasourceSettings(s backend.DataSourceInstanceSettings) (*rabbitmqclient.RabbitMQStreamClient, error) {
	client := rabbitmqclient.NewRabbitMQStreamClient()
	rabbitmqStreamOptions := rabbitmqclient.NewRabbitMQStreamOptions()

	log.DefaultLogger.Debug("Getting Datasource Settings from Client...")

	if err := json.Unmarshal(s.JSONData, rabbitmqStreamOptions); err != nil {
		return nil, err
	}

	log.DefaultLogger.Debug("Successfully unmarshelled the JSONData!")

	if password, exists := s.DecryptedSecureJSONData["password"]; exists {
		rabbitmqStreamOptions.Password = password
	}

	log.DefaultLogger.Debug("Successfully decrypted secure JSONData!")

	client.SetRabbitMQOptions(rabbitmqStreamOptions)

	log.DefaultLogger.Debug("Successfully set the RabbitMQOptions in the RabbitMQStreamClient!")

	return client, nil
}
