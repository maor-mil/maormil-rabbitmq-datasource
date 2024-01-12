package rabbitmqclient

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func failOnError(err error, msg string) error {
	if err != nil {
		log.DefaultLogger.Error(msg, "error", err)
	}
	return err
}
