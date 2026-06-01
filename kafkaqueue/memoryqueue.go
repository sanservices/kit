package kafkaqueue

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// ErrQueueFull is returned by Push when the queue has reached its maximum size.
var ErrQueueFull = errors.New("queue is full")

// MemoryQueue provides a bounded in-memory queue with buffering.
type MemoryQueue struct {
	in      chan interface{}
	out     chan interface{}
	buffer  []interface{}
	mu      sync.RWMutex
	closed  bool
	maxSize int
}

// NewMemoryQueue creates a new in-memory queue capped at maxSize items.
// maxSize <= 0 means unbounded (not recommended in production).
func NewMemoryQueue(maxSize int) *MemoryQueue {
	const initCapacity = 256

	in := make(chan interface{}, initCapacity)
	out := make(chan interface{}, initCapacity)

	mq := &MemoryQueue{
		in:      in,
		out:     out,
		buffer:  make([]interface{}, 0, initCapacity),
		maxSize: maxSize,
	}

	go mq.runBuffer(initCapacity)

	return mq
}

func (mq *MemoryQueue) runBuffer(initCapacity int) {
	defer close(mq.out)

loop:
	for {
		val, ok := <-mq.in
		if !ok {
			break loop
		}

		select {
		case mq.out <- val:
			continue
		default:
		}

		mq.mu.Lock()
		mq.buffer = append(mq.buffer, val)
		mq.mu.Unlock()

		for {
			mq.mu.RLock()
			bufLen := len(mq.buffer)
			mq.mu.RUnlock()

			if bufLen == 0 {
				break
			}

			slog.Debug("MemoryQueue buffer status", "buffer_size", bufLen, "out_channel_size", len(mq.out))

			select {
			case val, ok := <-mq.in:
				if !ok {
					break loop
				}
				mq.mu.Lock()
				mq.buffer = append(mq.buffer, val)
				mq.mu.Unlock()
			case mq.out <- mq.buffer[0]:
				mq.mu.Lock()
				mq.buffer = mq.buffer[1:]
				if len(mq.buffer) == 0 {
					mq.buffer = make([]interface{}, 0, initCapacity)
				}
				mq.mu.Unlock()
			}
		}
	}

	// Drain remaining buffer
	mq.mu.Lock()
	for len(mq.buffer) > 0 {
		mq.out <- mq.buffer[0]
		mq.buffer = mq.buffer[1:]
	}
	mq.mu.Unlock()
}

// Push adds a message to the memory queue. Returns ErrQueueFull when at capacity.
func (mq *MemoryQueue) Push(_ context.Context, q Queue) error {
	mq.mu.RLock()
	if mq.closed {
		mq.mu.RUnlock()
		return nil
	}
	mq.mu.RUnlock()

	if mq.maxSize > 0 && mq.Len() >= mq.maxSize {
		return ErrQueueFull
	}

	mq.in <- q
	return nil
}

// Pop removes and returns a message from the memory queue (blocking).
func (mq *MemoryQueue) Pop(ctx context.Context) (*Queue, bool) {
	select {
	case val, ok := <-mq.out:
		if !ok {
			return nil, false
		}
		q := val.(Queue)
		return &q, true
	case <-ctx.Done():
		return nil, false
	}
}

// TryPop attempts to pop a message without blocking.
func (mq *MemoryQueue) TryPop() (*Queue, bool) {
	select {
	case val, ok := <-mq.out:
		if !ok {
			return nil, false
		}
		q := val.(Queue)
		return &q, true
	default:
		return nil, false
	}
}

// Len returns the approximate total number of items across all internal buffers.
func (mq *MemoryQueue) Len() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return len(mq.buffer) + len(mq.out) + len(mq.in)
}

// Close closes the memory queue.
func (mq *MemoryQueue) Close() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if !mq.closed {
		mq.closed = true
		close(mq.in)
	}
}
