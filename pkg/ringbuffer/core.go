package ringbuffer

import (
	"errors"
	"sync"
)

// New returns a new RingBuffer whose buffer has the given size.
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	if size <= 0 {
		return nil
	}

	rb := &RingBuffer[T]{
		buf:  make([]T, size),
		size: size,
	}

	rb.logVerbose("New RingBuffer created with size: %d", size)
	return rb
}

// NewWithConfig creates a new RingBuffer with the given size and configuration.
// It returns an error if the size is less than or equal to 0.
func NewRingBufferWithConfig[T any](size int, config *RingBufferConfig) (*RingBuffer[T], error) {
	if size <= 0 {
		return nil, errors.New("size must be greater than 0")
	}

	rb := NewRingBuffer[T](size)

	rb.WithBlocking(config.Block)
	rb.logVerbose("Blocking mode set to: %v", config.Block)

	if config.RTimeout > 0 {
		rb.WithReadTimeout(config.RTimeout)
		rb.logVerbose("Read timeout set to: %v", config.RTimeout)
	}

	if config.WTimeout > 0 {
		rb.WithWriteTimeout(config.WTimeout)
		rb.logVerbose("Write timeout set to: %v", config.WTimeout)
	}

	rb.WithOverwrite(config.Overwrite)
	rb.logVerbose("Overwrite mode set to: %v", config.Overwrite)

	rb.WithVerbose(config.Verbose)
	rb.logVerbose("Verbose mode set to: %v", config.Verbose)

	return rb, nil
}

// Length returns the number of items in the buffer.
func (r *RingBuffer[T]) Length(locked bool) int {
	if !locked {
		r.mu.Lock()
		defer r.mu.Unlock()
	}

	if r.isFull {
		return r.size
	}

	if r.w >= r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

// Capacity returns the maximum number of items the buffer can hold.
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}

// Free returns the number of items that can be written to the buffer.
func (r *RingBuffer[T]) Free() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isFull {
		return 0
	}

	if r.w >= r.r {
		return r.size - r.w + r.r
	}

	return r.r - r.w
}

// IsFull returns whether the buffer is full.
func (r *RingBuffer[T]) IsFull() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isFull
}

// IsEmpty returns whether the buffer is empty.
func (r *RingBuffer[T]) IsEmpty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.isFull && r.r == r.w
}

// IsPaused returns whether the buffer is paused.
func (r *RingBuffer[T]) IsPaused() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

// GetBlockedReaders returns the number of readers currently blocked.
func (r *RingBuffer[T]) GetBlockedReaders() int {
	return int(r.blockedReaders.Load())
}

// GetBlockedWriters returns the number of writers currently blocked.
func (r *RingBuffer[T]) GetBlockedWriters() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int(r.blockedWriters.Load())
}

// CopyConfig copies the configuration from another RingBuffer.
func (r *RingBuffer[T]) CopyConfig(source *RingBuffer[T]) *RingBuffer[T] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.block = source.block
	r.rTimeout = source.rTimeout
	r.wTimeout = source.wTimeout
	r.overwrite = source.overwrite

	if r.block {
		r.readCond = sync.NewCond(&r.mu)
		r.writeCond = sync.NewCond(&r.mu)
	}

	return r
}
