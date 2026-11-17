package rabbitmqclient

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Passwords/users from issue #3 and friends: every URI delimiter, escapes, spaces and non-ASCII.
var trickyCredentials = []string{
	"guest",
	"p@ss",
	"p:ss",
	"p/ss",
	"p#ss",
	"p?ss",
	"p%ss",
	"p%41ss",
	"p&s=s+",
	"pa ss",
	`p"'\ss`,
	"pässwörd",
	"@:/#?%&=+ ",
}

func optionsWith(user, password, vhost string) *RabbitMQStreamOptions {
	opts := NewRabbitMQStreamOptions()
	opts.Host = "broker"
	opts.User, opts.Password, opts.VHost = user, password, vhost
	return opts
}

// streamClientURI replays what rabbitmq-stream-go-client v1.10.0 does with SetUser/SetPassword/SetVHost
// for every connection: Broker.GetUri() formats the URI, Client.connect() parses it, authenticates with the
// decoded userinfo and opens url.QueryUnescape(path) as the vhost.
func streamClientURI(t *testing.T, user, password, vhost string) (gotUser, gotPassword, gotVhost, host, port string) {
	t.Helper()
	uri := fmt.Sprintf("%s://%s:%s@%s:%s/%s", "rabbitmq-stream", user, password, "broker", "5552", vhost)
	u, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("the stream client would fail to parse %q: %v", uri, err)
	}
	gotPassword, _ = u.User.Password()
	gotVhost = "/"
	if len(u.Path) > 1 {
		gotVhost, _ = url.QueryUnescape(u.Path[1:])
	}
	return u.User.Username(), gotPassword, gotVhost, u.Hostname(), u.Port()
}

func TestStreamCredentialsRoundTripThroughTheStreamClientURI(t *testing.T) {
	for _, secret := range trickyCredentials {
		for _, vhost := range []string{"/", "my-vhost", "team/a", "with space", "ö?#"} {
			t.Run(fmt.Sprintf("%q@%q", secret, vhost), func(t *testing.T) {
				user, password, escapedVhost := streamCredentials(optionsWith("u"+secret, secret, vhost))

				gotUser, gotPassword, gotVhost, host, port := streamClientURI(t, user, password, escapedVhost)

				if host != "broker" || port != "5552" {
					t.Errorf("host/port = %q/%q, want broker/5552", host, port)
				}
				if gotUser != "u"+secret || gotPassword != secret {
					t.Errorf("credentials = %q/%q, want %q/%q", gotUser, gotPassword, "u"+secret, secret)
				}
				if gotVhost != vhost {
					t.Errorf("vhost = %q, want %q", gotVhost, vhost)
				}
			})
		}
	}
}

func TestAMQPConnectionKeepsCredentialsOutOfTheURI(t *testing.T) {
	for _, secret := range trickyCredentials {
		opts := optionsWith("user", secret, "/")
		address := amqpAddress(opts)

		if strings.Contains(address, "@") || (secret != "guest" && strings.Contains(address, secret)) {
			t.Errorf("credentials leaked into the AMQP address %q", address)
		}
		uri, err := amqp.ParseURI(address)
		if err != nil || uri.Host != "broker" || uri.Port != 5672 {
			t.Errorf("address %q does not parse to broker:5672: %+v, %v", address, uri, err)
		}

		config := amqpConfig(opts, nil)
		plain, ok := config.SASL[0].(*amqp.PlainAuth)
		if !ok || plain.Username != "user" || plain.Password != secret {
			t.Errorf("SASL PLAIN credentials = %+v, want user/%q", config.SASL[0], secret)
		}
		if config.Vhost != "/" {
			t.Errorf("vhost = %q, want /", config.Vhost)
		}
	}
}

// Documents issue #3: the URI the plugin used to build is broken by such passwords.
func TestTheFormerAMQPURIBreaksOnSpecialCharacters(t *testing.T) {
	for _, secret := range []string{"p/ss", "p#ss", "p?ss", "p%zz"} {
		former := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", "user", secret, "broker", 5672, "/")
		uri, err := amqp.ParseURI(former)
		if err == nil && uri.Host == "broker" && uri.Password == secret {
			t.Errorf("expected the former URI %q to be broken, got %+v", former, uri)
		}
	}
}

func TestIPv6HostsAreBracketed(t *testing.T) {
	opts := optionsWith("u", "p", "/")
	opts.Host = "::1"
	if got := amqpAddress(opts); got != "amqp://[::1]:5672/" {
		t.Errorf("amqpAddress = %q", got)
	}
}
