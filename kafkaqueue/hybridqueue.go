package kafkaqueue

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// HybridQueue provides a queue that uses Redis as primary storage
// and falls back to in-memory queue when Redis is unavailable.
type HybridQueue struct {
	redisQueue    *RedisQueue
	memoryQueue   *MemoryQueue
	useMemory     atomic.Bool
	mu            sync.RWMutex
	maxMemorySize int

	healthCheckInterval time.Duration
	stopHealthCheck     chan struct{}
}

// NewHybridQueue creates a new hybrid queue with Redis as primary and memory as fallback.
// maxMemorySize caps the in-memory fallback queue; use 0 for unbounded (not recommended).
func NewHybridQueue(redisQueue *RedisQueue, maxMemorySize int) *HybridQueue {
	hq := &HybridQueue{
		redisQueue:          redisQueue,
		memoryQueue:         NewMemoryQueue(maxMemorySize),
		maxMemorySize:       maxMemorySize,
		healthCheckInterval: 10 * time.Second,
		stopHealthCheck:     make(chan struct{}),
	}

	if redisQueue == nil || !redisQueue.IsEnabled() {
		hq.useMemory.Store(true)
		slog.Info("HybridQueue: starting with memory queue (Redis not available)")
	} else {
		hq.useMemory.Store(false)
		slog.Info("HybridQueue: starting with Redis queue")
	}

	go hq.healthCheck()

	return hq
}

func (hq *HybridQueue) healthCheck() {
	ticker := time.NewTicker(hq.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hq.checkRedisHealth()
		case <-hq.stopHealthCheck:
			return
		}
	}
}

func (hq *HybridQueue) checkRedisHealth() {
	if hq.redisQueue == nil || !hq.redisQueue.IsEnabled() {
		return
	}

	client := hq.redisQueue.Client()
	if client == nil {
		return
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(pingCtx).Err()

	if err != nil {
		if !hq.useMemory.Load() {
			hq.useMemory.Store(true)
			slog.Warn("HybridQueue: Redis unavailable, switching to memory queue", "error", err)
		}
	} else {
		if hq.useMemory.Load() {
			hq.drainMemoryToRedis(context.Background())
			hq.useMemory.Store(false)
			slog.Info("HybridQueue: Redis available, switching back to Redis queue")
		}
	}
}

func (hq *HybridQueue) drainMemoryToRedis(ctx context.Context) {
	count := 0
	for {
		msg, ok := hq.memoryQueue.TryPop()
		if !ok || msg == nil {
			break
		}

		if err := hq.redisQueue.Push(ctx, *msg); err != nil {
			hq.memoryQueue.Push(ctx, *msg)
			slog.Error("HybridQueue: failed to drain message to Redis, keeping in memory", "error", err)
			return
		}
		count++
	}

	if count > 0 {
		slog.Info("HybridQueue: drained messages from memory to Redis", "count", count)
	}
}

// Push adds a message to the queue (Redis if available, memory otherwise).
func (hq *HybridQueue) Push(ctx context.Context, q Queue) error {
	if hq.useMemory.Load() {
		return hq.memoryQueue.Push(ctx, q)
	}

	err := hq.redisQueue.Push(ctx, q)
	if err != nil {
		hq.useMemory.Store(true)
		slog.Warn("HybridQueue: Redis push failed, falling back to memory", "error", err)
		return hq.memoryQueue.Push(ctx, q)
	}

	return nil
}

// Pop removes and returns a message from the queue.
func (hq *HybridQueue) Pop(ctx context.Context, timeout time.Duration) (*Queue, error) {
	if hq.useMemory.Load() {
		msg, ok := hq.memoryQueue.Pop(ctx)
		if !ok {
			return nil, nil
		}
		return msg, nil
	}

	msg, err := hq.redisQueue.Pop(ctx, timeout)
	if err != nil {
		hq.useMemory.Store(true)
		slog.Warn("HybridQueue: Redis pop failed, falling back to memory", "error", err)

		memMsg, ok := hq.memoryQueue.Pop(ctx)
		if !ok {
			return nil, nil
		}
		return memMsg, nil
	}

	return msg, nil
}

// Len returns the current length of the active queue.
func (hq *HybridQueue) Len(ctx context.Context) (int64, error) {
	if hq.useMemory.Load() {
		return int64(hq.memoryQueue.Len()), nil
	}

	return hq.redisQueue.Len(ctx)
}

// Close closes both queues and stops the health check.
func (hq *HybridQueue) Close() error {
	close(hq.stopHealthCheck)
	hq.memoryQueue.Close()

	if hq.redisQueue != nil {
		return hq.redisQueue.Close()
	}
	return nil
}

// IsUsingMemory returns true if currently using the memory queue.
func (hq *HybridQueue) IsUsingMemory() bool {
	return hq.useMemory.Load()
}

// IsRedisEnabled returns true if Redis queue is configured and enabled.
func (hq *HybridQueue) IsRedisEnabled() bool {
	return hq.redisQueue != nil && hq.redisQueue.IsEnabled()
}
