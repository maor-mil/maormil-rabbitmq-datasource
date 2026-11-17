package plugin

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
)

const testUID = "abcdefghijklmnopqrstuvwxyz0123456789ABCD" // 40 chars, the maximum Grafana uid length

var livePathPattern = regexp.MustCompile(`^[A-z0-9_\-/=.]*$`)

func queryWith(json string) backend.DataQuery {
	return backend.DataQuery{RefID: "A", JSON: []byte(json)}
}

func pluginContext() backend.PluginContext {
	return backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: testUID}}
}

func channelOf(t *testing.T, ds *RabbitMQDatasource, json string) string {
	t.Helper()
	res := ds.query(context.Background(), pluginContext(), queryWith(json))
	if res.Error != nil {
		t.Fatalf("query %s: %v", json, res.Error)
	}
	return res.Frames[0].Meta.Channel
}

func TestEmptyQueriesKeepTheLegacyChannel(t *testing.T) {
	ds := &RabbitMQDatasource{}
	for _, q := range []string{``, `{}`, `{"refId":"A"}`, `{"source":"datasource"}`} {
		if got := channelOf(t, ds, q); got != "ds/"+testUID+"/"+STREAM_CHANNEL_PATH {
			t.Errorf("query %q -> channel %q, want the 1.x channel", q, got)
		}
	}
}

func TestSourcesRoundTripThroughTheChannelPath(t *testing.T) {
	ds := &RabbitMQDatasource{}
	sources := []rabbitmqclient.Source{
		{Type: rabbitmqclient.SourceStream, Stream: "orders", Offset: rabbitmqclient.OffsetNext},
		{Type: rabbitmqclient.SourceQueue, QueueMode: rabbitmqclient.QueueModeTap, Exchange: "amq.topic", RoutingKey: "sensors.#"},
		{Type: rabbitmqclient.SourceQueue, QueueMode: rabbitmqclient.QueueModeExisting, Queue: "jobs/ä #1"},
	}
	for _, source := range sources {
		path := ds.channelPath(testUID, source)
		if !livePathPattern.MatchString(path) || len("ds/"+testUID+"/"+path) > maxChannelLength {
			t.Errorf("invalid Live path %q", path)
		}
		got, err := ds.sourceFromPath(path)
		if err != nil || !reflect.DeepEqual(got, source) {
			t.Errorf("round trip of %+v: got %+v, %v", source, got, err)
		}
	}
}

func TestEqualSourcesShareOneChannel(t *testing.T) {
	ds := &RabbitMQDatasource{}
	a := channelOf(t, ds, `{"source":"queue","exchange":" amq.topic ","routingKey":"x","queue":"ignored in tap mode"}`)
	b := channelOf(t, ds, `{"source":"queue","queueMode":"tap","exchange":"amq.topic","routingKey":"x","stream":"ignored"}`)
	if a != b {
		t.Errorf("equivalent queries got different channels:\n%s\n%s", a, b)
	}
}

func TestLongSourcesUseARegistryReference(t *testing.T) {
	ds := &RabbitMQDatasource{}
	source := rabbitmqclient.Source{Type: rabbitmqclient.SourceStream, Stream: strings.Repeat("s", 200), Offset: rabbitmqclient.OffsetFirst}
	path := ds.channelPath(testUID, source)
	if !strings.HasPrefix(path, refPathPrefix) || len("ds/"+testUID+"/"+path) > maxChannelLength || !livePathPattern.MatchString(path) {
		t.Fatalf("unexpected path %q", path)
	}
	got, err := ds.sourceFromPath(path)
	if err != nil || !reflect.DeepEqual(got, source) {
		t.Fatalf("resolving %q: %+v, %v", path, got, err)
	}
	if _, err := (&RabbitMQDatasource{}).sourceFromPath(path); err != errUnknownSourceRef {
		t.Errorf("another instance cannot resolve the reference: got %v", err)
	}
}

func TestInvalidQueriesReturnAnError(t *testing.T) {
	ds := &RabbitMQDatasource{}
	for _, q := range []string{
		`{"source":"stream"}`,
		`{"source":"stream","stream":"s","offset":"middle"}`,
		`{"source":"queue","queueMode":"tap"}`,
		`{"source":"queue","queueMode":"existing"}`,
		`{"source":"queue","queueMode":"steal","queue":"q"}`,
		`{"source":"kafka"}`,
		`{not json`,
	} {
		if res := ds.query(context.Background(), pluginContext(), queryWith(q)); res.Error == nil {
			t.Errorf("query %s: expected an error", q)
		}
	}
}

func TestInvalidChannelPathsAreRejected(t *testing.T) {
	ds := &RabbitMQDatasource{}
	for _, path := range []string{"other", "src/!!!", "src/" + "e30", "ref/unknown"} {
		if _, err := ds.sourceFromPath(path); err == nil {
			t.Errorf("path %q: expected an error", path)
		}
	}
}
