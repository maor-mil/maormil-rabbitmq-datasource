# RabbitMQ Grafana Datasource Plugin

The RabbitMQ Streaming Datasource plugin for realtime data updates in [Grafana](https://grafana.com) Dashboards.

![Demo With Two RabbitMQs](img/demo_with_two_rabbitmqs.gif)

## What is RabbitMQ Stream?
If you are not fimiliar with RabbitMQ Stream Plugin, here are some good resources to read from about it:
* [RabbitMQ official page about streams](https://www.rabbitmq.com/streams.html)
* [RabbitMQ Stream Core vs Stream Plugin](https://www.rabbitmq.com/stream-core-plugin-comparison.html)
* [RabbitMQ Stream Practical Guide Part 1](https://www.cloudamqp.com/blog/rabbitmq-streams-and-replay-features-part-1-when-to-use-rabbitmq-streams.html)
* [RabbitMQ Stream Practical Guide Part 2](https://www.cloudamqp.com/blog/rabbitmq-streams-and-replay-features-part-2-how-to-work-with-rabbitmq-streams.html)
* [RabbitMQ Stream Practical Guide Part 3](https://www.cloudamqp.com/blog/rabbitmq-streams-and-replay-features-part-3-limits-and-configurations-for-streams-in-rabbitmq.html)

## Getting Started
### Reqirements

- RabbitMQ v3.12.10+ with `rabbitmq_stream` plugin enabled (should work on RabbitMQ v3.9+ but never was tested)
- Grafana v9.4.3+

> Note: This is a backend plugin, so the Grafana server should've access to the RabbitMQ broker.


### Add the RabbitMQ Datasource Plugin to your Grafana
#### Installation via grafana-cli tool

Use the grafana-cli tool to install the plugin from the commandline:

```bash
grafana-cli plugins install maor2475-rabbitmq-datasource
```

The plugin will be installed into your grafana plugins directory; the default is `/var/lib/grafana/plugins`. [More information on the cli tool](https://grafana.com/docs/grafana/latest/administration/cli/#plugins-commands).

#### Installation via zip file

Alternatively, you can manually download the [latest](https://github.com/maor-mil/maor2475-rabbitmq-datasource/releases/latest) release .zip file and unpack it into your grafana plugins directory; the default is `/var/lib/grafana/plugins`.


### Configuration Editor - Configure your RabbitMQ Stream
![New RabbitMQ Datasource](screenshots/rabbitmq_datasource.png)

[Add a data source](https://grafana.com/docs/grafana/latest/datasources/add-a-data-source/) by filling in the following fields (**TIP**: if you are not sure some field means just leave it with the default value)

#### Connection
Basic connection section to connect the RabbitMQ.

<img src="screenshots/new_rabbitmq_datasource/connection_section.png"
 alt="Connection Section" width="400"/>

| Field         | Type     | Is Required  | Default Value  | Description                                 |
|---------------|----------|--------------|----------------|---------------------------------------------|
| `Host`        | `string` | Yes          | `"localhost"`  | Hostname (or the IP) of the RabbitMQ server |
| `AMQP Port`   | `int`    | Yes          | `5672`         | The AMQP port of the RabbitMQ server        |
| `Stream Port` | `int`    | Yes      	  | `5552`         | The stream port of the RabbitMQ server      |
| `VHost`       | `string` | Yes       	  | `"/"`          | The virtual host the RabbitMQ server        |
---

#### Authentication
Basic authentication section to connect the RabbitMQ.

<img src="screenshots/new_rabbitmq_datasource/authentication_section.png"
 alt="Authentication Section" width="400"/>

| Field            | Type     | Is Required | Default Value | Description                                      |
|------------------|----------|-------------|---------------|--------------------------------------------------|
| `TLS Connection` | `bool`   | Yes         | `false`       | Should use TLS to connect to the RabbitMQ server |
| `Username`       | `string` | Yes         | `"guest"`     | Username to connect to the RabbitMQ server       |
| `Password`       | `string` | Yes         | -----         | Password to connect to the RabbitMQ server       |
---
 
#### Stream Settings
Stream decleration settings section, where you define the settings of your wanted
RabbitMQ stream and the settings of its consumer.

<img src="screenshots/new_rabbitmq_datasource/stream_settings_section.png"
 alt="Stream Settings Section" width="400"/>

| Field                    | Type     | Is Required | Default Value       | Description                                               |
|--------------------------|----------|-------------|---------------------|-----------------------------------------------------------|
| `Stream Name`            | `string` | Yes         | `""`                | The stream name that will be created                      |
| `Consumer Name`          | `string` | No          | `""`                | The consumer name that will be created                    |
| `Offset from Start`      | `bool`   | Yes         | `true`              | Should the consumer consume messages from the start or the end of the stored messages in the stream |
| `Max Age`                | `int`    | Yes         | `3,600,000,000,000` | The max age of messages in the stream in nano-seconds (set to 0 to disable the max-age limit) |
| `Max Length Bytes`       | `int`    | Yes         | `2,000,000,000`     | The max length of messages in bytes in the stream (set to 0 to disable the max-length-bytes limit) |
| `Max Segment Size Bytes` | `int`    | Yes         | `500,000,000`       | The max segment size in bytes in the stream               |
| `CRC`                    | `bool`   | Yes         | `false`             | When CRC control is disabled, the perfomance is increased |
---

#### Exchanges
Optional Section.
Array section for multiple exchanges that should be created in the RabbitMQ (they will also be recreated in case of connection lost to the RabbitMQ / Stream deletion in the RabbitMQ).
If the exchange already exists in the RabbitMQ, the plugin will not recreate the exchange but if the same exchange name already exists in the RabbitMQ with different settings that you gave, you might encounter some problems creating the RabbitMQ Datasource connections.

<img src="screenshots/new_rabbitmq_datasource/exchanges_section.png"
 alt="Exchanges Section" width="400"/>

| Field              | Type     | Is Required | Default Value              | Description                                              |
|--------------------|----------|-------------|----------------------------|----------------------------------------------------------|
| `Exchange Name`    | `string` | Yes         | `"Type the Exchange Name"` | The exchange name that should exist in the RabbitMQ      |
| `Exchange Type`    | `string` | Yes         | `fanout`                   | The exchange type (should only accept known RabbitMQ exchange types) |
| `Is Durable`       | `bool`   | Yes         | `true`                     | Should the exchange be durable                           |
| `Is Auto Deleted`  | `bool`   | Yes         | `false`                    | Should the exchange be auto deleted                      |
| `Is Internal`      | `bool`   | Yes         | `false`                    | Should the exchange be internal                          |
| `Is NoWait`        | `bool`   | Yes         | `false`                    | Should the exchange be noWait                            |
---

#### Bindings
Optional Section.
Array section for multiple bindings that should be created in the RabbitMQ (they will also be recreated in case of connection lost to the RabbitMQ / Stream deletion in the RabbitMQ).
If the binding already exists in the RabbitMQ, the plugin will not recreate the binding.
There is a support of binding of exchange to queue (or a stream) type of binding AND binding of exchange to exchange.

<img src=screenshots/new_rabbitmq_datasource/bindings_section.png
 alt="Bindings Section" width="400"/>

| Field              | Type     | Is Required | Default Value                                          | Description                                              |
|--------------------|----------|-------------|--------------------------------------------------------|----------------------------------------------------------|
| `Is Queue Binding` | `bool`   | Yes         | `true`                                                 | Should binding be from Exchange to queue/stream (if disabled, the binding will be from exchange to exchange) |
| `Sender Name`      | `string` | Yes         | `"Some Exchange"`                                      | The exchange to bind from |
| `Routing Key`      | `string` | Yes         | `"/"`                                                  | The routing key to bind between the sender exchange and the receiver |
| `Receiver Name`    | `string` | Yes         | `"Probably your stream name or some another exchange"` | The stream/queue/exchange to bind to |
| `Is No Wait`       | `bool`   | Yes         | `false`                                                | Should binding be noWait |
---

## Query Editor
<img src=screenshots/rabbitmq_query_editor.png
 alt="Bindings Section" width="300"/>

Please pay attention that there is no real query editor in the RabbitMQ plugin.
Once you set the datasource settings you ready to go.
You can still change the time range query in the default Grafana query editor which will impact what data is being showen and how fast the query interval is.
This plugin was planned and deisgned to work with the [Plotly by nline](https://github.com/nline/nline-plotlyjs-panel) panel plugin.
Feel free to check this awesome plugin (and if you wish not to use plotly you can still use transformations together with this RabbitMQ plugin).

## Reconnecting to the RabbitMQ
The plugin handle most chaos scenarios automatically:
* If the stream is deleted in the RabbitMQ itself - the stream, its consumer, and the pre-configured exchanges and bindings will be recreated.
* If the Grafana is down once it goes up again, the consumer will be recreated.
* If the RabbitMQ is down the plugin will try to keep reconnecting to the RabbitMQ (it will only stop if the datasource is deleted by the user).

### Important Note about the Deletion of RabbitMQ Datasource
**If the user decides to remove the RabbitMQ Datasource (from the Grafana) - ONLY THE CONSUMER will be removed from the RabbitMQ.** The stream, exchanges and bindings that were created from the RabbitMQ Datasource will still exists in the RabbitMQ. So if you wish for them to be deleted, you must do it manually.

## Known limitations
* This plugin does not support advanced TLS Configuration for RabbitMQ connections.
* This plugin only supports JSON based messages that and throws away any non JSON message. The JSON can contain numbers, strings, booleans, and JSON formatted values. Nested object values can be extracted using the Extract Fields transformation (or being processed by the [Plotly by nline](https://github.com/nline/nline-plotlyjs-panel) panel plugin).
* **This plugin automatically attaches timestamps to the messages** when they are received. Timestamps included in the message body can be parsed using the Convert field type transformation. **The key name of the added timestamp is: `RmqMsgConsumedTimestamp`**
* Currently the only supported offsets are First and Last (no specific offset or timestamp support when consuming messages from the RabbitMQ streams).
* As written above - **If the user decides to remove the RabbitMQ Datasource (from the Grafana) - ONLY THE CONSUMER will be removed from the RabbitMQ.** The stream, exchanges and bindings that were created from the RabbitMQ Datasource will still exists in the RabbitMQ. So if you wish for them to be deleted, you must do it manually.

## Known Errors and Causes When Trying To Connect To RabbitMQ

* `dial tcp <ip/hostname>:<port>: connect: connection refused`
   * There is no RabbitMQ up with the given `<ip/hostname>`.
   * Wrong `Stream Port`.

<img src="screenshots/known_errors/connection_refused.png"
 alt="Exchanges Section" width="400" height="40"/>

* `timeout 10000 ms - waiting Code, operation: commandPeerProperties`
    * The Stream Plugin is not enabled in the RabbitMQ.
    * Wrong `AMQP Port`.
    * Wrong `VHost`.

<img src="screenshots/known_errors/timeout_error.png"
 alt="Exchanges Section" width="400" height="40"/>

* There could be more errors, but the rest of them are pretty intuitive.

## Credits
### Creating the actual Streaming Backend Plugin
The streaming feature in Grafana is still experimental and the documntation to how to create a streaming backend datasource is lacking.
There are not a lot of plugins that support this feature yet, so in order to create this RabbitMQ Datasource plugin I read and used some parts from all of the existing Grafana Streaming Datasources:
* [MQTT Datasource (by GrafanaLabs)](https://github.com/grafana/mqtt-datasource/)
* [Kafka Datasource (by hamedkarbasi93)](https://github.com/hoptical/grafana-kafka-datasource)
* [Websocket Backend Datasource Example (by GrafanaLabs)](https://github.com/grafana/grafana-plugin-examples/tree/main/examples/datasource-streaming-backend-websocket)
* [Websocket Datasource (by Golioth)](https://github.com/golioth/grafana-websocket-plugin)

Speical credit for the framer.go which was copied from the MQTT Datasource and modified a little bit.

### Communication with the RabbitMQ
The plugin is using 2 main core packages to be able to communicate with the RabbitMQ, both by the [RabbitMQ core team](https://github.com/rabbitmq):
* [amqp091-go](https://github.com/rabbitmq/amqp091-go)
* [rabbitmq-stream-go-client](https://github.com/rabbitmq/rabbitmq-stream-go-client)

## Contributing

Thank you for considering contributing! If you find an issue or have a better way to do something, feel free to open an issue or a PR.

## License

This repository is open-sourced software licensed under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).