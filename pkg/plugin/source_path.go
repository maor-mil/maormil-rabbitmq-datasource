package plugin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
)

// Grafana Live channels look like "ds/<datasource uid>/<path>"; the whole channel is limited to 160
// characters and the path to [A-Za-z0-9_\-/=.] (grafana-plugin-sdk-go/live).
const (
	maxChannelLength = 160
	// sourcePathPrefix + base64url(JSON source): self-contained, so any Grafana instance can run the stream.
	sourcePathPrefix = "src/"
	// refPathPrefix + hash: fallback for sources too long for the channel; resolved from the registry that
	// QueryData filled on this plugin instance.
	refPathPrefix = "ref/"
)

var errUnknownSourceRef = errors.New("unknown stream reference, refresh the panel")

// channelPath returns the Live channel path that identifies source. Panels with equal (normalised) sources
// share one channel and therefore one RabbitMQ consumer.
func (ds *RabbitMQDatasource) channelPath(datasourceUID string, source rabbitmqclient.Source) string {
	if source.Type == rabbitmqclient.SourceDatasource {
		return STREAM_CHANNEL_PATH // unchanged since 1.x, so existing dashboards keep their channel
	}
	encoded, _ := json.Marshal(source)
	path := sourcePathPrefix + base64.RawURLEncoding.EncodeToString(encoded)
	if len("ds/"+datasourceUID+"/"+path) <= maxChannelLength {
		return path
	}
	sum := sha256.Sum256(encoded)
	path = refPathPrefix + hex.EncodeToString(sum[:16])
	ds.sources.Store(path, source)
	return path
}

// sourceFromPath is the inverse of channelPath.
func (ds *RabbitMQDatasource) sourceFromPath(path string) (rabbitmqclient.Source, error) {
	var source rabbitmqclient.Source
	switch {
	case path == STREAM_CHANNEL_PATH, path == "":
		return rabbitmqclient.Source{Type: rabbitmqclient.SourceDatasource}, nil
	case strings.HasPrefix(path, sourcePathPrefix):
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(path, sourcePathPrefix))
		if err != nil {
			return source, fmt.Errorf("invalid stream path: %w", err)
		}
		if err := json.Unmarshal(raw, &source); err != nil {
			return source, fmt.Errorf("invalid stream path: %w", err)
		}
	case strings.HasPrefix(path, refPathPrefix):
		stored, ok := ds.sources.Load(path)
		if !ok {
			return source, errUnknownSourceRef
		}
		source = stored.(rabbitmqclient.Source)
	default:
		return source, fmt.Errorf("unknown stream path %q", path)
	}
	source = source.Normalize()
	if source.Type == rabbitmqclient.SourceDatasource {
		// The data source's own stream only has the legacy path; one canonical channel per consumer.
		return source, fmt.Errorf("invalid stream path %q", path)
	}
	return source, source.Validate()
}
