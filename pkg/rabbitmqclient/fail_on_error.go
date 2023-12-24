package rabbitmqclient

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func failOnError(err error, msg string) error {
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("%s: %s\n", msg, err))
	}
	return err
}
