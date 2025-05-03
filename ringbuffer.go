package ringbuffer

import (
	"io"
	"sync"
	"time"

	"github.com/AlexsanderHamir/ringbuffer/config"
	"github.com/AlexsanderHamir/ringbuffer/errors"
)

// RingBuffer is a circular buffer that implements io.ReaderWriter interface.
// It operates like a buffered pipe, where data is written to a RingBuffer
// and can be read back from another goroutine.
// It is safe to concurrently read and write RingBuffer.
//
// Key features:
// - Thread-safe concurrent read/write operations
// - Configurable blocking/non-blocking behavior
// - Timeout support for read/write operations
// - Pre-read hook for custom blocking behavior
// - Efficient circular buffer implementation
type RingBuffer[T any] struct {
	buf       []T
	size      int
	r         int // next position to read
	w         int // next position to write
	isFull    bool
	err       error
	block     bool
	rTimeout  time.Duration // Applies to writes (waits for the read condition)
	wTimeout  time.Duration // Applies to read (wait for the write condition)
	mu        sync.Mutex
	readCond  *sync.Cond // Signaled when data has been read.
	writeCond *sync.Cond // Signaled when data has been written.

	blockedReaders int
	blockedWriters int

	// Hook function that will be called before blocking on a read or hitting a deadline
	// Returns true if the hook successfully handled the situation, false otherwise
	preReadBlockHook func() bool

	// Hook function that will be called before blocking on a write or hitting a deadline
	// Returns true if the hook successfully handled the situation, false otherwise
	preWriteBlockHook func() bool
}

// New returns a new RingBuffer whose buffer has the given size.
func New[T any](size int) *RingBuffer[T] {
	if size <= 0 {
		return nil
	}

	return &RingBuffer[T]{
		buf:  make([]T, size),
		size: size,
	}
}

// NewWithConfig creates a new RingBuffer with the given size and configuration.
// It returns an error if the size is less than or equal to 0.
func NewWithConfig[T any](size int, cfg *config.RingBufferConfig) (*RingBuffer[T], error) {
	if size <= 0 {
		return nil, errors.ErrInvalidLength
	}

	rb := New[T](size)
	rb.WithBlocking(cfg.Block)

	if cfg.RTimeout > 0 {
		rb.WithReadTimeout(cfg.RTimeout)
	}

	if cfg.WTimeout > 0 {
		rb.WithWriteTimeout(cfg.WTimeout)
	}

	if cfg.PreReadBlockHook != nil {
		rb.WithPreReadBlockHook(cfg.PreReadBlockHook)
	}

	return rb, nil
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
// This method automatically enables blocking mode since timeouts require blocking behavior.
func (r *RingBuffer[T]) WithTimeout(d time.Duration) *RingBuffer[T] {
	if d > 0 {
		r.WithBlocking(true)
	}

	r.mu.Lock()
	r.rTimeout = d
	r.wTimeout = d
	r.mu.Unlock()
	return r
}

// WithReadTimeout sets the timeout for read operations.
// Read operations wait for writes to complete, so this sets the write timeout.
// This method automatically enables blocking mode since timeouts require blocking behavior.
func (r *RingBuffer[T]) WithReadTimeout(d time.Duration) *RingBuffer[T] {
	if d > 0 && !r.block {
		r.WithBlocking(true)
	}
	r.mu.Lock()
	r.wTimeout = d
	r.mu.Unlock()
	return r
}

// WithWriteTimeout sets the timeout for write operations.
// Write operations wait for reads to complete, so this sets the read timeout.
// This method automatically enables blocking mode since timeouts require blocking behavior.
func (r *RingBuffer[T]) WithWriteTimeout(d time.Duration) *RingBuffer[T] {
	if d > 0 && !r.block {
		r.WithBlocking(true)
	}

	r.mu.Lock()
	r.rTimeout = d
	r.mu.Unlock()

	return r
}

// WithPreReadBlockHook sets a hook function that will be called before blocking on a read
// or hitting a deadline. This allows for custom handling of blocking situations,
// such as trying alternative sources for data.
func (r *RingBuffer[T]) WithPreReadBlockHook(hook func() bool) *RingBuffer[T] {
	r.mu.Lock()
	r.preReadBlockHook = hook
	r.mu.Unlock()
	return r
}

// WithPreWriteBlockHook sets a hook function that will be called before blocking on a write
// or hitting a deadline. This allows for custom handling of blocking situations,
// such as trying alternative destinations for data.
func (r *RingBuffer[T]) WithPreWriteBlockHook(hook func() bool) *RingBuffer[T] {
	r.mu.Lock()
	r.preWriteBlockHook = hook
	r.mu.Unlock()
	return r
}

// Length returns the number of items that can be read.
// This is the actual number of items in the buffer.
func (r *RingBuffer[T]) Length(lock bool) int {
	if !lock {
		r.mu.Lock()
		defer r.mu.Unlock()
	}

	if r.err == io.EOF {
		return 0
	}

	if r.isFull {
		return r.size
	}

	if r.w >= r.r {
		return r.w - r.r
	}
	return r.size - r.r + r.w
}

// Capacity returns the size of the underlying buffer
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}

// Free returns the number of items that can be written without blocking.
// This is the available space in the buffer.
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

// IsFull returns true when the ringbuffer is full.
func (r *RingBuffer[T]) IsFull() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isFull
}

// IsEmpty returns true when the ringbuffer is empty.
func (r *RingBuffer[T]) IsEmpty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return !r.isFull && r.w == r.r
}

// GetBlockedWriters returns the number of blocked writers
func (r *RingBuffer[T]) GetBlockedWriters() int {
	if r.err == io.EOF {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.blockedWriters
}

// CopyConfig copies the configuration settings from the source buffer to the target buffer.
// This includes blocking mode, timeouts, and cancellation context.
func (r *RingBuffer[T]) CopyConfig(source *RingBuffer[T]) *RingBuffer[T] {
	r.WithBlocking(source.block)

	if source.rTimeout > 0 {
		r.WithReadTimeout(source.rTimeout)
	}

	if source.wTimeout > 0 {
		r.WithWriteTimeout(source.wTimeout)
	}

	r.WithPreReadBlockHook(source.preReadBlockHook)

	return r
}

// ClearBuffer clears all items in the buffer and resets read/write positions.
// Useful when shrinking the buffer or cleaning up resources.
func (r *RingBuffer[T]) ClearBuffer() {
	var zero T
	if r.w > r.r {
		for i := r.r; i < r.w; i++ {
			r.buf[i] = zero
		}
	} else {
		for i := r.r; i < r.size; i++ {
			r.buf[i] = zero
		}
		for i := range r.w {
			r.buf[i] = zero
		}
	}

	r.r = 0
	r.w = 0
	r.isFull = false
}

// Close closes the ring buffer and cleans up resources.
// Behavior:
// - Sets error to io.EOF
// - Clears all items in the buffer
// - Signals all waiting readers and writers
// - All subsequent operations will return io.EOF
func (r *RingBuffer[T]) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err == io.EOF {
		return nil
	}

	r.setErr(io.EOF, true)
	r.ClearBuffer()

	if r.block {
		r.readCond.Broadcast()
		r.writeCond.Broadcast()
	}

	return nil
}
