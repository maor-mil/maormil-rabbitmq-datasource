package rabbitmqclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type testPKI struct{ caPEM, certPEM, keyPEM string }

// newTestPKI creates an in-memory CA and a client certificate signed by it.
func newTestPKI(t *testing.T) testPKI {
	t.Helper()
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "client"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, caTemplate, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	encode := func(kind string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der}))
	}
	return testPKI{encode("CERTIFICATE", caDER), encode("CERTIFICATE", der), encode("EC PRIVATE KEY", keyDER)}
}

func tlsOptions(mutate func(o *RabbitMQStreamOptions)) *RabbitMQStreamOptions {
	o := NewRabbitMQStreamOptions()
	o.Host = "broker.example"
	o.IsTLS = true
	if mutate != nil {
		mutate(o)
	}
	return o
}

func TestBuildTLSConfigIsNilWithoutTLS(t *testing.T) {
	config, err := buildTLSConfig(tlsOptions(func(o *RabbitMQStreamOptions) { o.IsTLS = false }))
	if err != nil || config != nil {
		t.Fatalf("got %v, %v; want nil, nil", config, err)
	}
}

func TestBuildTLSConfigDefaults(t *testing.T) {
	config, err := buildTLSConfig(tlsOptions(nil))
	if err != nil {
		t.Fatal(err)
	}
	if config.MinVersion != tls.VersionTLS12 || config.ServerName != "broker.example" || config.InsecureSkipVerify ||
		config.RootCAs != nil || len(config.Certificates) != 0 {
		t.Fatalf("unexpected default TLS config: %+v", config)
	}
}

func TestBuildTLSConfigOptions(t *testing.T) {
	pki := newTestPKI(t)
	config, err := buildTLSConfig(tlsOptions(func(o *RabbitMQStreamOptions) {
		o.MinVersion = TLSVersion13
		o.ServerName = "rabbit.internal"
		o.TLSSkipVerify = true
		o.TLSAuthWithCACert, o.CACert = true, pki.caPEM
		o.TLSAuth, o.ClientCert, o.ClientKey = true, pki.certPEM, pki.keyPEM
	}))
	if err != nil {
		t.Fatal(err)
	}
	if config.MinVersion != tls.VersionTLS13 || config.ServerName != "rabbit.internal" || !config.InsecureSkipVerify {
		t.Errorf("version/server name/skip verify not applied: %+v", config)
	}
	if config.RootCAs == nil || len(config.Certificates) != 1 {
		t.Errorf("CA or client certificate not applied")
	}
}

func TestBuildTLSConfigErrors(t *testing.T) {
	pki := newTestPKI(t)
	other := newTestPKI(t)
	cases := map[string]struct {
		mutate func(o *RabbitMQStreamOptions)
		want   string
	}{
		"unknown min version":   {func(o *RabbitMQStreamOptions) { o.MinVersion = "1.0" }, "unsupported minimum version"},
		"CA enabled but empty":  {func(o *RabbitMQStreamOptions) { o.TLSAuthWithCACert = true }, "no CA certificate"},
		"CA is not PEM":         {func(o *RabbitMQStreamOptions) { o.TLSAuthWithCACert, o.CACert = true, "garbage" }, "not a valid PEM"},
		"client auth, no key":   {func(o *RabbitMQStreamOptions) { o.TLSAuth, o.ClientCert = true, pki.certPEM }, "missing"},
		"mismatched key":        {func(o *RabbitMQStreamOptions) { o.TLSAuth, o.ClientCert, o.ClientKey = true, pki.certPEM, other.keyPEM }, "invalid client certificate"},
		"EXTERNAL without cert": {func(o *RabbitMQStreamOptions) { o.SASLMechanism = SASLMechanismExternal }, "requires \"TLS Client Auth\""},
		"EXTERNAL without TLS": {func(o *RabbitMQStreamOptions) {
			o.IsTLS = false
			o.SASLMechanism = SASLMechanismExternal
		}, "requires a TLS connection"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := buildTLSConfig(tlsOptions(tc.mutate))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestValidateSASLMechanism(t *testing.T) {
	for _, ok := range []string{"", SASLMechanismPlain, SASLMechanismExternal} {
		if err := validateSASLMechanism(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	if err := validateSASLMechanism("SCRAM-SHA-256"); err == nil {
		t.Error("expected an error for an unsupported mechanism")
	}
}

func TestAMQPUsesTLSAndExternalAuth(t *testing.T) {
	options := tlsOptions(func(o *RabbitMQStreamOptions) { o.SASLMechanism = SASLMechanismExternal })
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	if got := amqpAddress(options); got != "amqps://broker.example:5672/" {
		t.Errorf("amqpAddress = %q, want the amqps scheme", got)
	}
	config := amqpConfig(options, tlsConfig)
	if config.TLSClientConfig != tlsConfig {
		t.Error("TLS config not passed to amqp091")
	}
	if _, ok := config.SASL[0].(*amqp.ExternalAuth); !ok {
		t.Errorf("SASL = %T, want *amqp.ExternalAuth", config.SASL[0])
	}
}
