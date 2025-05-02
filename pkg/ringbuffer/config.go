package ringbuffer

import (
	"sync"
	"time"
)

// IsBlocking returns whether the buffer is configured to block on operations
func (c *RingBufferConfig) IsBlocking() bool {
	return c.Block
}

// GetReadTimeout returns the read timeout duration
func (c *RingBufferConfig) GetReadTimeout() time.Duration {
	return c.RTimeout
}

// GetWriteTimeout returns the write timeout duration
func (c *RingBufferConfig) GetWriteTimeout() time.Duration {
	return c.WTimeout
}

// IsVerbose returns whether verbose logging is enabled
func (c *RingBufferConfig) IsVerbose() bool {
	return c.Verbose
}

// WithBlocking sets the blocking mode of the ring buffer.
// When blocking is enabled:
// - Read operations will block when the buffer is empty
// - Write operations will block when the buffer is full
// - Condition variables are created for synchronization
func (r *RingBuffer[T]) WithBlocking(block bool) *RingBuffer[T] {
	r.mu.Lock()
	r.block = block
	if block {
		r.readCond = sync.NewCond(&r.mu)
		r.writeCond = sync.NewCond(&r.mu)
	}
	r.mu.Unlock()
	return r
}

// WithTimeout sets both read and write timeouts for the ring buffer.
// When a timeout occurs, the operation returns context.DeadlineExceeded.
// A timeout of 0 or less disables timeouts.
func (r *RingBuffer[T]) WithTimeout(d time.Duration) *RingBuffer[T] {
	r.mu.Lock()
	r.rTimeout = d
	r.wTimeout = d
	r.mu.Unlock()
	return r
}

// WithReadTimeout sets the timeout for read operations.
// Read operations wait for writes to complete, so this sets the write timeout.
func (r *RingBuffer[T]) WithReadTimeout(d time.Duration) *RingBuffer[T] {
	r.mu.Lock()
	// Read operations wait for writes to complete,
	// therefore we set the wTimeout.
	r.wTimeout = d
	r.mu.Unlock()
	return r
}

// WithWriteTimeout sets the timeout for write operations.
// Write operations wait for reads to complete, so this sets the read timeout.
func (r *RingBuffer[T]) WithWriteTimeout(d time.Duration) *RingBuffer[T] {
	r.mu.Lock()
	// Write operations wait for reads to complete,
	// therefore we set the rTimeout.
	r.rTimeout = d
	r.mu.Unlock()
	return r
}

// WithOverwrite sets whether the buffer should overwrite data when full
func (r *RingBuffer[T]) WithOverwrite(overwrite bool) *RingBuffer[T] {
	r.mu.Lock()
	r.overwrite = overwrite
	r.mu.Unlock()
	return r
}

// WithVerbose sets whether verbose logging should be enabled
func (r *RingBuffer[T]) WithVerbose(verbose bool) *RingBuffer[T] {
	r.mu.Lock()
	r.verbose = verbose
	r.mu.Unlock()
	return r
}
