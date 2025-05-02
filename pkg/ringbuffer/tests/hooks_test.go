package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteHooks(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	var readCount, writeCount int
	var readMu, writeMu sync.Mutex

	// Add read hook
	rb.WithOnReadHook(func(item int, success bool) {
		if success {
			readMu.Lock()
			readCount++
			readMu.Unlock()
		}
	})

	// Add write hook
	rb.WithOnWriteHook(func(item int, success bool) {
		if success {
			writeMu.Lock()
			writeCount++
			writeMu.Unlock()
		}
	})

	t.Run("hooks on write", func(t *testing.T) {
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)

		writeMu.Lock()
		assert.Equal(t, 2, writeCount)
		writeMu.Unlock()
	})

	t.Run("hooks on read", func(t *testing.T) {
		_, err := rb.GetOne()
		require.NoError(t, err)
		_, err = rb.GetOne()
		require.NoError(t, err)

		readMu.Lock()
		assert.Equal(t, 2, readCount)
		readMu.Unlock()
	})
}

func TestBlockingOperations(t *testing.T) {
	tests := []struct {
		name    string
		config  *ringbuffer.RingBufferConfig
		timeout time.Duration
		wantErr bool
	}{
		{
			name: "blocking enabled",
			config: &ringbuffer.RingBufferConfig{
				Block: true,
			},
			timeout: 100 * time.Millisecond,
			wantErr: false,
		},
		{
			name: "blocking disabled",
			config: &ringbuffer.RingBufferConfig{
				Block: false,
			},
			timeout: 100 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := ringbuffer.NewRingBufferWithConfig[int](2, tt.config)
			require.NoError(t, err)
			require.NotNil(t, rb)

			// Fill the buffer
			err = rb.Write(1)
			require.NoError(t, err)
			err = rb.Write(2)
			require.NoError(t, err)

			// Try to write to full buffer
			done := make(chan error)
			go func() {
				done <- rb.Write(3)
			}()

			select {
			case err := <-done:
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(tt.timeout):
				if !tt.wantErr {
					t.Error("write operation timed out")
				}
			}
		})
	}
}

func TestOverwriteMode(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](3)
	require.NotNil(t, rb)

	rb.WithOverwrite(true)

	t.Run("overwrite enabled", func(t *testing.T) {
		// Fill the buffer
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)
		err = rb.Write(3)
		require.NoError(t, err)

		// Write to full buffer with overwrite
		err = rb.Write(4)
		require.NoError(t, err)

		// Verify the oldest item was overwritten
		val, err := rb.GetOne()
		require.NoError(t, err)
		assert.Equal(t, 2, val) // 1 was overwritten
	})
}

func TestPauseResume(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	t.Run("pause and resume", func(t *testing.T) {
		// Write some data
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)

		// Pause the buffer
		rb.Pause()
		assert.True(t, rb.IsPaused())

		// Try to write while paused
		err = rb.Write(3)
		require.NoError(t, err) // Should succeed but not actually write

		// Verify length hasn't changed
		assert.Equal(t, 2, rb.Length(false))

		// Resume the buffer
		rb.Resume()
		assert.False(t, rb.IsPaused())

		// Write should work again
		err = rb.Write(3)
		require.NoError(t, err)
		assert.Equal(t, 3, rb.Length(false))
	})
}

func TestFlushAndReset(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	t.Run("flush and reset", func(t *testing.T) {
		// Write some data
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)
		err = rb.Write(3)
		require.NoError(t, err)

		// Flush the buffer
		rb.Flush()
		assert.True(t, rb.IsEmpty())
		assert.Equal(t, 0, rb.Length(false))

		// Write more data
		err = rb.Write(4)
		require.NoError(t, err)
		err = rb.Write(5)
		require.NoError(t, err)

		// Reset the buffer
		rb.Reset()
		assert.True(t, rb.IsEmpty())
		assert.Equal(t, 0, rb.Length(false))
		assert.False(t, rb.IsPaused())
	})
}
