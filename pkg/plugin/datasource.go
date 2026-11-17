package plugin

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"

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

	// Connect eagerly so the topology is declared as soon as the data source is used, but do not fail the
	// instance: CheckHealth reports the connection error ("Save & test") and RunStream retries.
	if _, err = client.Connect(); err != nil {
		log.DefaultLogger.Warn("Could not connect to RabbitMQ yet", "RabbitMQ Stream", client.ToString(), "error", err)
	} else {
		log.DefaultLogger.Debug("Successfully connected to the RabbitMQ!")
	}

	return NewRabbitMQDatasource(client), nil
}

type RabbitMQDatasource struct {
	Client rabbitmqclient.Client
	// sources resolves "ref/<hash>" channel paths of sources too long to encode in the channel itself.
	sources sync.Map
}

func NewRabbitMQDatasource(client rabbitmqclient.Client) *RabbitMQDatasource {
	return &RabbitMQDatasource{
		Client: client,
	}
}

// Dispose is called by the SDK when the data source settings change (and the instance is replaced by a
// new one) or the instance is evicted. It only closes the connections: the plugin never deletes streams,
// exchanges or bindings in RabbitMQ, since Grafana does not tell plugins whether a data source was deleted
// or merely saved.
func (ds *RabbitMQDatasource) Dispose() {
	ds.Client.Close()
}

func getDatasourceSettings(s backend.DataSourceInstanceSettings) (*rabbitmqclient.RabbitMQStreamClient, error) {
	client := rabbitmqclient.NewRabbitMQStreamClient()
	rabbitmqStreamOptions := rabbitmqclient.NewRabbitMQStreamOptions()

	log.DefaultLogger.Debug("Getting Datasource Settings from Client...")

	if err := json.Unmarshal(s.JSONData, rabbitmqStreamOptions); err != nil {
		return nil, err
	}

	log.DefaultLogger.Debug("Successfully unmarshelled the JSONData!")

	secure := s.DecryptedSecureJSONData
	if password, exists := secure["password"]; exists {
		rabbitmqStreamOptions.Password = password
	}
	// TLS material is secret and only ever read from secureJsonData.
	rabbitmqStreamOptions.CACert = secure["tlsCACert"]
	rabbitmqStreamOptions.ClientCert = secure["tlsClientCert"]
	rabbitmqStreamOptions.ClientKey = secure["tlsClientKey"]

	log.DefaultLogger.Debug("Successfully decrypted secure JSONData!")

	client.SetRabbitMQOptions(rabbitmqStreamOptions)

	log.DefaultLogger.Debug("Successfully set the RabbitMQOptions in the RabbitMQStreamClient!")

	return client, nil
}
