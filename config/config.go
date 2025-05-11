package config

import "time"

// RingBufferConfig holds the configuration for a RingBuffer
type RingBufferConfig[T any] struct {
	Block             bool
	RTimeout          time.Duration
	WTimeout          time.Duration
	PreReadBlockHook  func() (obj T, tryAgain bool, success bool)
	PreWriteBlockHook func() bool
}

// IsBlocking returns whether the buffer is in blocking mode
func (c *RingBufferConfig[T]) IsBlocking() bool {
	return c.Block
}

// GetReadTimeout returns the read timeout duration
func (c *RingBufferConfig[T]) GetReadTimeout() time.Duration {
	return c.RTimeout
}

// GetWriteTimeout returns the write timeout duration
func (c *RingBufferConfig[T]) GetWriteTimeout() time.Duration {
	return c.WTimeout
}

// GetPreReadBlockHook returns the pre-read block hook function
func (c *RingBufferConfig[T]) GetPreReadBlockHook() func() (obj T, tryAgain bool, success bool) {
	return c.PreReadBlockHook
}

// GetPreWriteBlockHook returns the pre-write block hook function
func (c *RingBufferConfig[T]) GetPreWriteBlockHook() func() bool {
	return c.PreWriteBlockHook
}
