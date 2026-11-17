package plugin

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// CheckHealth reports whether the plugin can reach RabbitMQ with the current settings. When the instance is
// not connected (e.g. the broker was down when it started, or no panel is streaming) it connects, which
// also (re-)declares the configured topology.
func (ds *RabbitMQDatasource) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if !ds.Client.IsConnected() {
		if _, err := ds.Client.Connect(); err != nil {
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: fmt.Sprintf("Cannot connect to RabbitMQ: %v", err),
			}, nil
		}
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "RabbitMQ Connected",
	}, nil
}
