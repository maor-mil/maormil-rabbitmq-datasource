package plugin

import "time"

type TimestampedMessage struct {
	Timestamp time.Time
	Value     []byte
}

func NewTimestampedMessage(value []byte) *TimestampedMessage {
	return &TimestampedMessage{
		Timestamp: time.Now(),
		Value:     value,
	}
}
