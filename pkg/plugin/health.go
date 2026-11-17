package plugin

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// CheckHealth reports whether the plugin can reach RabbitMQ and declare the configured topology with the
// current settings. Connect reuses an open connection and (re-)declares the stream, exchanges and bindings.
func (ds *RabbitMQDatasource) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if _, err := ds.Client.Connect(); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Cannot connect to RabbitMQ: %v", err),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "RabbitMQ Connected",
	}, nil
}
