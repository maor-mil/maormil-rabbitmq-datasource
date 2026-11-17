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

	FilterValues    []string `json:"filterValues"`
	MatchUnfiltered bool     `json:"matchUnfiltered"`
	SQL             string   `json:"sql"`
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

		FilterValues:    q.FilterValues,
		MatchUnfiltered: q.MatchUnfiltered,
		SQL:             q.SQL,
	}.Normalize()
}

func (ds *RabbitMQDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Debug("Started QueryData method!")
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		response.Responses[q.RefID] = ds.query(ctx, req.PluginContext, q)
	}

	return response, nil
}

// query returns an empty frame whose Live channel identifies what to stream; the data then arrives through
// RunStream on that channel.
func (ds *RabbitMQDatasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
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
	if err := ds.validateSQLFilter(ctx, source); err != nil {
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

// validateSQLFilter asks the broker once per distinct filter whether it accepts the SQL expression, so a typo
// shows up on the panel instead of making the stream fail silently in the background.
func (ds *RabbitMQDatasource) validateSQLFilter(ctx context.Context, source rabbitmqclient.Source) error {
	if source.Type != rabbitmqclient.SourceStream || source.SQL == "" {
		return nil
	}
	key, _ := json.Marshal(source)
	if _, ok := ds.validFilters.Load(string(key)); ok {
		return nil
	}
	if err := ds.Client.ValidateStreamFilter(ctx, source); err != nil {
		return err
	}
	ds.validFilters.Store(string(key), true)
	return nil
}
