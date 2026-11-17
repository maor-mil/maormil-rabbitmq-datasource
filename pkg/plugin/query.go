package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/live"
	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
)

// queryModel is the panel query (src/types.ts RabbitMQQuery). An empty query streams the data source's own
// stream, exactly like before per-query sources existed.
type queryModel struct {
	Source     rabbitmqclient.SourceType `json:"source"`
	Stream     string                    `json:"stream"`
	Offset     rabbitmqclient.Offset     `json:"offset"`
	QueueMode  rabbitmqclient.QueueMode  `json:"queueMode"`
	Queue      string                    `json:"queue"`
	Exchange   string                    `json:"exchange"`
	RoutingKey string                    `json:"routingKey"`
}

func (q queryModel) source() rabbitmqclient.Source {
	return rabbitmqclient.Source{
		Type:       q.Source,
		Stream:     q.Stream,
		Offset:     q.Offset,
		QueueMode:  q.QueueMode,
		Queue:      q.Queue,
		Exchange:   q.Exchange,
		RoutingKey: q.RoutingKey,
	}.Normalize()
}

func (ds *RabbitMQDatasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Debug("Started QueryData method!")
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		response.Responses[q.RefID] = ds.query(req.PluginContext, q)
	}

	return response, nil
}

// query returns an empty frame whose Live channel identifies what to stream; the data then arrives through
// RunStream on that channel.
func (ds *RabbitMQDatasource) query(pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var model queryModel
	if len(query.JSON) > 0 {
		if err := json.Unmarshal(query.JSON, &model); err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("invalid query: %v", err))
		}
	}
	source := model.source()
	if err := source.Validate(); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
	}

	uid := pCtx.DataSourceInstanceSettings.UID
	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: uid,
		Path:      ds.channelPath(uid, source),
	}
	frame := data.NewFrame(FRAME_NAME)
	frame.SetMeta(&data.FrameMeta{Channel: channel.String()})

	return backend.DataResponse{Frames: data.Frames{frame}}
}
