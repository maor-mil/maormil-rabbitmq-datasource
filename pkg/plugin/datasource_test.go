package plugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/maormil/rabbitmq-datasource/pkg/rabbitmqclient"
)

func settingsFrom(jsonData string, secure map[string]string) backend.DataSourceInstanceSettings {
	return backend.DataSourceInstanceSettings{JSONData: []byte(jsonData), DecryptedSecureJSONData: secure}
}

func TestGetDatasourceSettingsAppliesDefaultsToEmptySettings(t *testing.T) {
	client, err := getDatasourceSettings(settingsFrom(`{}`, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := client.RabbitMQOptions
	want := rabbitmqclient.NewRabbitMQStreamOptions()

	if got.Host != want.Host || got.AmqpPort != want.AmqpPort || got.StreamPort != want.StreamPort || got.VHost != want.VHost {
		t.Errorf("connection defaults not applied: got %+v", got)
	}
	if got.User != rabbitmqclient.DefaultUser || got.Password != rabbitmqclient.DefaultPassword {
		t.Errorf("credential defaults not applied: user=%q password=%q", got.User, got.Password)
	}
	if got.RequestedHeartbeat != rabbitmqclient.DefaultRequestedHeartbeat || got.RequestedMaxFrameSize != rabbitmqclient.DefaultRequestedMaxFrameSize ||
		got.WriteBuffer != rabbitmqclient.DefaultWriteBuffer || got.ReadBuffer != rabbitmqclient.DefaultReadBuffer {
		t.Errorf("advanced defaults not applied: got %+v", got)
	}
	if got.StreamOptions == nil {
		t.Fatal("stream options are nil")
	}
	if *got.StreamOptions != *want.StreamOptions {
		t.Errorf("stream defaults not applied: got %+v, want %+v", *got.StreamOptions, *want.StreamOptions)
	}
}

func TestGetDatasourceSettingsKeepsConfiguredValues(t *testing.T) {
	client, err := getDatasourceSettings(settingsFrom(
		`{"host":"broker","streamPort":5551,"username":"svc","streamOptions":{"streamName":"orders","maxAge":0,"offsetFromStart":false}}`,
		map[string]string{"password": "s3cr@t:/#?"},
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := client.RabbitMQOptions

	if got.Host != "broker" || got.StreamPort != 5551 || got.User != "svc" || got.Password != "s3cr@t:/#?" {
		t.Errorf("configured connection values not kept: %+v", got)
	}
	if got.AmqpPort != rabbitmqclient.DefaultAmqpPort {
		t.Errorf("missing amqpPort should default to %d, got %d", rabbitmqclient.DefaultAmqpPort, got.AmqpPort)
	}
	so := got.StreamOptions
	if so.StreamName != "orders" || so.MaxAge != 0 || so.OffsetFromStart {
		t.Errorf("configured stream values not kept (an explicit 0/false must not be replaced by a default): %+v", *so)
	}
	if so.MaxLengthBytes != rabbitmqclient.DefaultMaxLengthBytes || so.MaxSegmentSizeBytes != rabbitmqclient.DefaultMaxSegmentSizeBytes {
		t.Errorf("missing stream values should default: %+v", *so)
	}
}

func TestGetDatasourceSettingsRejectsInvalidJSON(t *testing.T) {
	if _, err := getDatasourceSettings(settingsFrom(`{not json`, nil)); err == nil {
		t.Fatal("expected an error for invalid jsonData")
	}
}

func TestGetDatasourceSettingsReadsTLSSecretsOnlyFromSecureJSONData(t *testing.T) {
	client, err := getDatasourceSettings(settingsFrom(
		`{"tlsConnection":true,"tlsAuth":true,"tlsAuthWithCACert":true,"serverName":"rabbit.internal","tlsMinVersion":"1.3","saslMechanism":"EXTERNAL","tlsCACert":"must be ignored"}`,
		map[string]string{"tlsCACert": "CA", "tlsClientCert": "CERT", "tlsClientKey": "KEY"},
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	o := client.RabbitMQOptions
	if !o.IsTLS || !o.TLSAuth || !o.TLSAuthWithCACert || o.ServerName != "rabbit.internal" || o.MinVersion != "1.3" || o.SASLMechanism != "EXTERNAL" {
		t.Errorf("TLS settings not parsed: %+v", o.TLSOptions)
	}
	if o.CACert != "CA" || o.ClientCert != "CERT" || o.ClientKey != "KEY" {
		t.Errorf("TLS secrets must come from secureJsonData: %q %q %q", o.CACert, o.ClientCert, o.ClientKey)
	}
}

func TestGetDatasourceSettingsTLSDefaults(t *testing.T) {
	client, _ := getDatasourceSettings(settingsFrom(`{}`, nil))
	if o := client.RabbitMQOptions; o.SASLMechanism != rabbitmqclient.SASLMechanismPlain || o.MinVersion != rabbitmqclient.TLSVersion12 || o.IsTLS {
		t.Errorf("unexpected TLS defaults: sasl=%q min=%q tls=%v", o.SASLMechanism, o.MinVersion, o.IsTLS)
	}
}
