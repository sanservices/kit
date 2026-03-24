package kafkaqueue

import (
	"context"
	"encoding/json"
	"time"
)

// Queue represents a message in the queue.
type Queue struct {
	Topic string
	Msg   interface{}
	Ctx   context.Context
	Time  time.Time
}

// QueueMessage is the JSON-serializable representation of a Queue for Redis storage.
type QueueMessage struct {
	Topic string          `json:"topic"`
	Msg   json.RawMessage `json:"msg"`
	Time  time.Time       `json:"time"`
}
