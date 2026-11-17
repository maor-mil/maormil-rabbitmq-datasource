package rabbitmqclient

import (
	"crypto/tls"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Credentials and the vhost are user input and may contain any character (`@ : / # ? % & = +` ...).
// Neither client may receive them inside a hand-built URI (issue #3).

// amqpAddress is the AMQP 0.9.1 address without credentials or vhost; both are passed through amqpConfig.
// TLS ("amqps") is only used when the scheme says so: amqp091-go ignores TLSClientConfig for "amqp".
func amqpAddress(options *RabbitMQStreamOptions) string {
	scheme := "amqp"
	if options.IsTLS {
		scheme = "amqps"
	}
	return scheme + "://" + net.JoinHostPort(options.Host, strconv.Itoa(options.AmqpPort)) + "/"
}

// amqpConfig authenticates with the raw credentials (SASL PLAIN) or the TLS client certificate (SASL
// EXTERNAL) and selects the raw vhost, so nothing needs to be escaped. A non-nil SASL list and a non-empty
// Vhost override whatever the URI contains.
func amqpConfig(options *RabbitMQStreamOptions, tlsConfig *tls.Config) amqp.Config {
	var auth amqp.Authentication = &amqp.PlainAuth{Username: options.User, Password: options.Password}
	if options.SASLMechanism == SASLMechanismExternal {
		auth = &amqp.ExternalAuth{}
	}
	return amqp.Config{
		SASL:            []amqp.Authentication{auth},
		Vhost:           options.VHost,
		TLSClientConfig: tlsConfig,
		Heartbeat:       10 * time.Second,
		Locale:          "en_US",
	}
}

// streamCredentials returns the user, password and vhost encoded for rabbitmq-stream-go-client.
//
// For every connection it opens (the locator and each consumer/producer connection) the client builds
// "scheme://user:password@host:port/vhost" with fmt.Sprintf from these values, url.Parses it, and then
// authenticates with the *decoded* userinfo and opens url.QueryUnescape(path) as the vhost
// (Broker.GetUri / Client.connect in v1.10.0). Percent-encoding the values beforehand makes that URI parse
// back to the original values.
//
// Known upstream limitation: a vhost containing '+' or '%' cannot round-trip through the client's
// QueryUnescape of the already-decoded path.
func streamCredentials(options *RabbitMQStreamOptions) (user, password, vhost string) {
	return escapeURIComponent(options.User), escapeURIComponent(options.Password), url.PathEscape(options.VHost)
}

// escapeURIComponent percent-encodes every byte that is not an RFC 3986 "unreserved" character.
func escapeURIComponent(value string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0F])
	}
	return b.String()
}
