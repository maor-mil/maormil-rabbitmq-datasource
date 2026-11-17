package rabbitmqclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
)

// SASL mechanisms supported for both the AMQP 0.9.1 and the stream connection.
const (
	SASLMechanismPlain    = "PLAIN"
	SASLMechanismExternal = "EXTERNAL"
)

// Minimum TLS versions that can be configured.
const (
	TLSVersion12 = "1.2"
	TLSVersion13 = "1.3"
)

// TLSOptions are the TLS settings of the data source. The jsonData keys follow Grafana's standard TLS
// settings (tlsAuth, tlsAuthWithCACert, tlsSkipVerify, serverName, and secureJsonData tlsCACert,
// tlsClientCert, tlsClientKey), so they look familiar to Grafana users and provisioning files.
type TLSOptions struct {
	// TLSAuth presents a client certificate (mutual TLS).
	TLSAuth bool `json:"tlsAuth"`
	// TLSAuthWithCACert verifies the broker with TLSCACert instead of the system root CAs.
	TLSAuthWithCACert bool `json:"tlsAuthWithCACert"`
	// TLSSkipVerify disables verification of the broker certificate. Only for testing.
	TLSSkipVerify bool `json:"tlsSkipVerify"`
	// ServerName overrides the name used to verify the broker certificate (SNI). Defaults to the host.
	ServerName string `json:"serverName"`
	// MinVersion is "1.2" (default) or "1.3".
	MinVersion string `json:"tlsMinVersion"`

	// PEM content from secureJsonData (never from jsonData).
	CACert     string `json:"-"`
	ClientCert string `json:"-"`
	ClientKey  string `json:"-"`
}

// buildTLSConfig returns the TLS configuration for both connections, or nil when TLS is disabled.
func buildTLSConfig(options *RabbitMQStreamOptions) (*tls.Config, error) {
	if !options.IsTLS {
		if options.SASLMechanism == SASLMechanismExternal {
			return nil, errors.New("EXTERNAL authentication requires a TLS connection with a client certificate")
		}
		return nil, nil
	}
	t := options.TLSOptions

	minVersion, err := tlsMinVersion(t.MinVersion)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		MinVersion: minVersion,
		ServerName: t.ServerName,
		// #nosec G402 -- only when the user explicitly enables "Skip TLS Verify" (testing setups).
		InsecureSkipVerify: t.TLSSkipVerify,
	}
	if config.ServerName == "" {
		config.ServerName = options.Host
	}

	if t.TLSAuthWithCACert {
		if strings.TrimSpace(t.CACert) == "" {
			return nil, errors.New("TLS: \"With CA Cert\" is enabled but no CA certificate is configured")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(t.CACert)) {
			return nil, errors.New("TLS: the CA certificate is not a valid PEM certificate")
		}
		config.RootCAs = pool
	}

	if t.TLSAuth {
		if strings.TrimSpace(t.ClientCert) == "" || strings.TrimSpace(t.ClientKey) == "" {
			return nil, errors.New("TLS: \"TLS Client Auth\" is enabled but the client certificate or key is missing")
		}
		certificate, err := tls.X509KeyPair([]byte(t.ClientCert), []byte(t.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("TLS: invalid client certificate or key: %w", err)
		}
		config.Certificates = []tls.Certificate{certificate}
	} else if options.SASLMechanism == SASLMechanismExternal {
		return nil, errors.New("EXTERNAL authentication requires \"TLS Client Auth\" with a client certificate")
	}

	return config, nil
}

func tlsMinVersion(version string) (uint16, error) {
	switch version {
	case "", TLSVersion12:
		return tls.VersionTLS12, nil
	case TLSVersion13:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("TLS: unsupported minimum version %q (use %q or %q)", version, TLSVersion12, TLSVersion13)
	}
}

func validateSASLMechanism(mechanism string) error {
	switch mechanism {
	case "", SASLMechanismPlain, SASLMechanismExternal:
		return nil
	default:
		return fmt.Errorf("unsupported authentication mechanism %q (use %q or %q)", mechanism, SASLMechanismPlain, SASLMechanismExternal)
	}
}
