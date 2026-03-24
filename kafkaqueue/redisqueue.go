package kafkaqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConfig holds configuration for the Redis queue.
type RedisConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
	// QueuePrefix is the prefix for the Redis list key.
	// The final key will be "{QueuePrefix}:{hostname}".
	QueuePrefix string `json:"queue_prefix" yaml:"queue_prefix"`
}

// Deserializer converts a topic name and raw JSON into a typed message.
// If nil is provided to NewRedisQueue, messages are returned as json.RawMessage.
type Deserializer func(topic string, raw json.RawMessage) (interface{}, error)

// RedisQueue implements a message queue using Redis List operations.
type RedisQueue struct {
	client       *redis.Client
	queueKey     string
	enabled      bool
	deserializer Deserializer
}

// NewRedisQueue creates a new Redis queue client.
// The deserializer is optional — if nil, popped messages will have Msg as json.RawMessage.
func NewRedisQueue(cfg RedisConfig, deserializer Deserializer) (*RedisQueue, error) {
	if !cfg.Enabled {
		slog.Info("RedisQueue: disabled")
		return &RedisQueue{enabled: false}, nil
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	prefix := cfg.QueuePrefix
	if prefix == "" {
		prefix = "queue"
	}
	queueKey := fmt.Sprintf("%s:%s", prefix, host)

	slog.Info("RedisQueue: connected", "addr", addr, "queue_key", queueKey)

	return &RedisQueue{
		client:       client,
		queueKey:     queueKey,
		enabled:      true,
		deserializer: deserializer,
	}, nil
}

// Push adds a message to the Redis queue (RPUSH).
func (r *RedisQueue) Push(ctx context.Context, q Queue) error {
	if !r.enabled {
		return fmt.Errorf("redis queue is disabled")
	}

	msgBytes, err := json.Marshal(q.Msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message for topic %s: %w", q.Topic, err)
	}

	msg := QueueMessage{
		Topic: q.Topic,
		Msg:   msgBytes,
		Time:  q.Time,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message for topic %s: %w", q.Topic, err)
	}

	if err := r.client.RPush(context.Background(), r.queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis queue for topic %s: %w", q.Topic, err)
	}

	return nil
}

// Pop removes and returns a message from the Redis queue (BLPOP).
// Returns nil, nil if no message is available within the timeout.
func (r *RedisQueue) Pop(ctx context.Context, timeout time.Duration) (*Queue, error) {
	if !r.enabled {
		return nil, fmt.Errorf("redis queue is disabled")
	}

	result, err := r.client.BLPop(ctx, timeout, r.queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop from Redis queue: %w", err)
	}

	if len(result) < 2 {
		return nil, nil
	}

	var msg QueueMessage
	if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queue message: %w", err)
	}

	q := &Queue{
		Topic: msg.Topic,
		Ctx:   ctx,
		Time:  msg.Time,
	}

	if r.deserializer != nil {
		typedMsg, err := r.deserializer(msg.Topic, msg.Msg)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize message for topic %s: %w", msg.Topic, err)
		}
		q.Msg = typedMsg
	} else {
		q.Msg = msg.Msg
	}

	return q, nil
}

// Len returns the current length of the Redis queue.
func (r *RedisQueue) Len(ctx context.Context) (int64, error) {
	if !r.enabled {
		return 0, nil
	}
	return r.client.LLen(ctx, r.queueKey).Result()
}

// Close closes the Redis client connection.
func (r *RedisQueue) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// IsEnabled returns whether the Redis queue is enabled.
func (r *RedisQueue) IsEnabled() bool {
	return r.enabled
}

// Client returns the underlying Redis client for health checks.
func (r *RedisQueue) Client() *redis.Client {
	return r.client
}
