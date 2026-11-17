package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/message"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func CheckErr(err error) {
	if err != nil {
		fmt.Printf("%s ", err)
		os.Exit(1)
	}
}

var RABBITMQ_HOST = "localhost"
var RABBITMQ_STREAM_PORT = 5552
var RABBITMQ_USER = "guest"
var RABBITMQ_PASSWORD = "guest"

// Set SHOULD_CREATE_STRAEM to true if your RabbitQM Datasource didn't already create the stream
var SHOULD_CREATE_STRAEM = false
var RABBITMQ_STREAM_NAME = "rabbitmq.stream"
var RABBITMQ_STREAM_MB_BYTE_CAPACITY int64 = 500

var STREAM_PUBLISHER_NAME = "demo.publisher"
var GENERATED_NUMBERS_MINIMUM_VALUE = 100
var GENERATED_NUMBERS_MAXIMUM_VALUE = 20000

// Try a panel with Source "Stream", Filter values "emea", or SQL filter "region IN ('apac', 'amer')".
var REGIONS = []string{"emea", "apac", "amer"}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Stream Publisher for checking the RabbitMQ Datasource Plugin")
	fmt.Println("You can configure the consts above the main function")
	fmt.Println("Connecting to RabbitMQ streaming ...")

	env, err := stream.NewEnvironment(
		stream.NewEnvironmentOptions().
			SetHost(RABBITMQ_HOST).
			SetPort(RABBITMQ_STREAM_PORT).
			SetUser(RABBITMQ_USER).
			SetPassword(RABBITMQ_PASSWORD).
			SetAddressResolver(stream.AddressResolver{
				Host: RABBITMQ_HOST,
				Port: RABBITMQ_STREAM_PORT,
			}),
	)
	CheckErr(err)

	streamName := RABBITMQ_STREAM_NAME
	if SHOULD_CREATE_STRAEM {
		err = env.DeclareStream(streamName,
			&stream.StreamOptions{
				MaxLengthBytes: stream.ByteCapacity{}.MB(RABBITMQ_STREAM_MB_BYTE_CAPACITY),
			},
		)
		CheckErr(err)
	}

	// Each message carries a "region" application property (usable in SQL filter expressions) and the same
	// value as its stream filter value (usable as a Bloom "Filter values" entry). Filtering needs RabbitMQ 3.13+
	// (Bloom) / 4.2+ (SQL); older brokers ignore the filter value.
	producer, err := env.NewProducer(streamName, stream.NewProducerOptions().SetFilter(
		stream.NewProducerFilter(func(m message.StreamMessage) string {
			return fmt.Sprint(m.GetApplicationProperties()["region"])
		}),
	))
	CheckErr(err)

	go func() {
		for {
			generated_number := rand.Intn(
				GENERATED_NUMBERS_MAXIMUM_VALUE-GENERATED_NUMBERS_MINIMUM_VALUE,
			) + GENERATED_NUMBERS_MINIMUM_VALUE
			region := REGIONS[rand.Intn(len(REGIONS))]
			message_to_send := fmt.Sprintf("{\"name\": \"%s\", \"value\": %d, \"region\": \"%s\"}", STREAM_PUBLISHER_NAME, generated_number, region)
			msg := amqp.NewMessage([]byte(message_to_send))
			msg.ApplicationProperties = map[string]any{"region": region}
			err := producer.Send(msg)
			CheckErr(err)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	fmt.Println("Press any key to stop ")
	_, _ = reader.ReadString('\n')
	err = producer.Close()
	CheckErr(err)

}
