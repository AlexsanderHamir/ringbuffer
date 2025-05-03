package config

import "time"

// RingBufferConfig holds the configuration for a RingBuffer
type RingBufferConfig struct {
	Block             bool
	RTimeout          time.Duration
	WTimeout          time.Duration
	PreReadBlockHook  func() bool
	PreWriteBlockHook func() bool
}

// IsBlocking returns whether the buffer is in blocking mode
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

// GetPreReadBlockHook returns the pre-read block hook function
func (c *RingBufferConfig) GetPreReadBlockHook() func() bool {
	return c.PreReadBlockHook
}

// GetPreWriteBlockHook returns the pre-write block hook function
func (c *RingBufferConfig) GetPreWriteBlockHook() func() bool {
	return c.PreWriteBlockHook
}
