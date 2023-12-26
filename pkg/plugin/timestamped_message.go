package plugin

import "time"

type TimestampedMessage struct {
	Timestamp time.Time
	Value     []byte
}
