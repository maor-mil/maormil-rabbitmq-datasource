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
	log.DefaultLogger.Info("Started QueryData method!")
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := ds.query(req.PluginContext, q)

		response.Responses[q.RefID] = res
	}

	log.DefaultLogger.Info("Finished QueryData method!")

	return response, nil
}

func (ds *RabbitMQDatasource) query(pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("Started query method!")
	response := backend.DataResponse{}
	var qm QueryModel
	response.Error = json.Unmarshal(query.JSON, &qm)

	if response.Error != nil {
		return response
	}

	frame := data.NewFrame(FRAME_NAME)

	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: pCtx.DataSourceInstanceSettings.UID,
		Path:      STREAM_PATH,
	}
	frame.SetMeta(&data.FrameMeta{Channel: channel.String()})

	response.Frames = append(response.Frames, frame)

	log.DefaultLogger.Info("Finished query method!")

	return response
}
