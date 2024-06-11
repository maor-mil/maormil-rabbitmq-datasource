# Changelog

## 1.0.4-unsigned (Unreleased)

* Uupdate exchanges and bindings to use only one connection and channel when created/removed.
* Change the names of the default exchange and default binding.
* Change the plugin name from maor2475-rabbitmq-datasource to maormil-rabbitmq-datasource

## 1.0.3-unsigned (Unreleased)

* Update @grafana/create-plugin frontend dependencies and grafana-plugin-sdk-go backend dependencies.

## 1.0.2-unsigned (Unreleased)

* Fixed 'Plugin unavailable' issue.
* Changed some log messages format in the backend.
* Changed query instruction from pressing 'Refresh Query' to 'Refresh Dashboard'.
* The user can now choose to delete stream/exchanges/bindings when the RabbitMQ Datasource is deleted.
* Changed the Exchange Type selection from free text, to strict select between all the RabbitMQ exchange type options.

## 1.0.1-unsigned (Unreleased)

* Adding value attibutes to fields in ConfigEditor and changing logs format and log levels in the backend.
* Update README and screenshots to the new ConfigEditor
* Fixed jsonData of edited datasource to actually contains the previous state of the datasource.

## 1.0.0-unsigned (Unreleased)

* Initial release as unsigned plugin.
