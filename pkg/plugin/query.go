package plugin

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/live"
)

func (ds *RabbitMQDatasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Debug("Started QueryData method!")
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := ds.query(req.PluginContext, q)

		response.Responses[q.RefID] = res
	}

	log.DefaultLogger.Debug("Finished QueryData method!")

	return response, nil
}

func (ds *RabbitMQDatasource) query(pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Debug("Started query method!")
	response := backend.DataResponse{}
	var qm QueryModel
	response.Error = json.Unmarshal(query.JSON, &qm)

	if response.Error != nil {
		return response
	}

	frame := data.NewFrame("rabbitmq")

	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: pCtx.DataSourceInstanceSettings.UID,
		Path:      "rabbitmq",
	}
	frame.SetMeta(&data.FrameMeta{Channel: channel.String()})

	response.Frames = append(response.Frames, frame)

	log.DefaultLogger.Debug("Finished query method!")

	return response
}
