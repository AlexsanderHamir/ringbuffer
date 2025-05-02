package ringbuffer

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrTooMuchDataToWrite is returned when the data to write is more than the buffer size.
	ErrTooMuchDataToWrite = errors.New("too much data to write")

	// ErrTooMuchDataToRead is returned when trying to read more data than available.
	ErrTooMuchDataToRead = errors.New("too much data to read")

	// ErrIsFull is returned when the buffer is full and not blocking.
	ErrIsFull = errors.New("ringbuffer is full")

	// ErrIsEmpty is returned when the buffer is empty and not blocking.
	ErrIsEmpty = errors.New("ringbuffer is empty")

	// ErrIsNotEmpty is returned when the buffer is not empty and not blocking.
	ErrIsNotEmpty = errors.New("ringbuffer is not empty")

	// ErrAcquireLock is returned when the lock is not acquired on Try operations.
	ErrAcquireLock = errors.New("unable to acquire lock")
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
	overwrite bool
	verbose   bool
	rTimeout  time.Duration // Applies to writes (waits for the read condition)
	wTimeout  time.Duration // Applies to read (wait for the write condition)
	mu        sync.Mutex
	readCond  *sync.Cond // Signaled when data has been read.
	writeCond *sync.Cond // Signaled when data has been written.

	blockedReaders atomic.Uint32
	blockedWriters atomic.Uint32
	paused         bool // Whether the buffer is paused

	// Hook functions for various buffer events
	preReadBlockHook  func() bool                   // Called before blocking on read
	preWriteBlockHook func() bool                   // Called before blocking on write
	onFullHook        func()                        // Called when buffer becomes full
	onEmptyHook       func()                        // Called when buffer becomes empty
	onWriteHook       func(item T, success bool)    // Called after write attempt
	onReadHook        func(item T, success bool)    // Called after read attempt
	onResetHook       func()                        // Called after buffer reset
	onFlushHook       func()                        // Called after buffer flush
	onCloseHook       func()                        // Called before buffer close
	onErrorHook       func(err error)               // Called when error occurs
	onStateChangeHook func(isFull bool, length int) // Called when buffer state changes
	onPauseHook       func()                        // Called when buffer is paused
	onResumeHook      func()                        // Called when buffer is resumed
}

// RingBufferConfig holds configuration options for a RingBuffer
type RingBufferConfig struct {
	Block     bool
	RTimeout  time.Duration
	WTimeout  time.Duration
	Overwrite bool
	Verbose   bool
}
