package plugin

import (
	"context"
	"encoding/json"
	"fmt"

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
	log.DefaultLogger.Info("Done QueryData method!")
	return response, nil
}

func (ds *RabbitMQDatasource) query(pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("Started query method!")
	response := backend.DataResponse{}
	var qm QueryModel
	response.Error = json.Unmarshal(query.JSON, &qm)
	log.DefaultLogger.Info(fmt.Sprintf("qm: %+v", qm))

	if response.Error != nil {
		return response
	}

	frame := data.NewFrame("")

	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: pCtx.DataSourceInstanceSettings.UID,
		Path:      ds.Client.GetUniqueClientId(),
	}
	frame.SetMeta(&data.FrameMeta{Channel: channel.String()})

	response.Frames = append(response.Frames, frame)

	log.DefaultLogger.Info("Done query method!")
	return response
}
