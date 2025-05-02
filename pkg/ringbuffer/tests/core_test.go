package tests

import (
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		wantNil  bool
		capacity int
	}{
		{
			name:     "valid size",
			size:     10,
			wantNil:  false,
			capacity: 10,
		},
		{
			name:     "zero size",
			size:     0,
			wantNil:  true,
			capacity: 0,
		},
		{
			name:     "negative size",
			size:     -1,
			wantNil:  true,
			capacity: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := ringbuffer.NewRingBuffer[int](tt.size)
			if tt.wantNil {
				assert.Nil(t, rb)
			} else {
				require.NotNil(t, rb)
				assert.Equal(t, tt.capacity, rb.Capacity())
				assert.True(t, rb.IsEmpty())
				assert.False(t, rb.IsFull())
			}
		})
	}
}

func TestRingBufferBasicOperations(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	t.Run("initial state", func(t *testing.T) {
		assert.Equal(t, 5, rb.Capacity())
		assert.Equal(t, 0, rb.Length(false))
		assert.Equal(t, 5, rb.Free())
		assert.True(t, rb.IsEmpty())
		assert.False(t, rb.IsFull())
	})

	t.Run("write and read", func(t *testing.T) {
		// Write some values
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)
		err = rb.Write(3)
		require.NoError(t, err)

		assert.Equal(t, 3, rb.Length(false))
		assert.Equal(t, 2, rb.Free())
		assert.False(t, rb.IsEmpty())
		assert.False(t, rb.IsFull())

		// Read values
		val, err := rb.GetOne()
		require.NoError(t, err)
		assert.Equal(t, 1, val)

		val, err = rb.GetOne()
		require.NoError(t, err)
		assert.Equal(t, 2, val)

		assert.Equal(t, 1, rb.Length(false))
		assert.Equal(t, 4, rb.Free())
	})
}

func TestRingBufferFullEmpty(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](3)
	require.NotNil(t, rb)

	t.Run("fill buffer", func(t *testing.T) {
		err := rb.Write(1)
		require.NoError(t, err)
		err = rb.Write(2)
		require.NoError(t, err)
		err = rb.Write(3)
		require.NoError(t, err)

		assert.True(t, rb.IsFull())
		assert.Equal(t, 3, rb.Length(false))
		assert.Equal(t, 0, rb.Free())
	})

	t.Run("empty buffer", func(t *testing.T) {
		_, err := rb.GetOne()
		require.NoError(t, err)
		_, err = rb.GetOne()
		require.NoError(t, err)
		_, err = rb.GetOne()
		require.NoError(t, err)

		assert.True(t, rb.IsEmpty())
		assert.Equal(t, 0, rb.Length(false))
		assert.Equal(t, 3, rb.Free())
	})
}

func TestRingBufferWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		config  *ringbuffer.RingBufferConfig
		wantErr bool
	}{
		{
			name: "valid config",
			size: 5,
			config: &ringbuffer.RingBufferConfig{
				Block:     true,
				RTimeout:  time.Second,
				WTimeout:  time.Second,
				Overwrite: false,
			},
			wantErr: false,
		},
		{
			name:    "invalid size",
			size:    0,
			config:  &ringbuffer.RingBufferConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := ringbuffer.NewRingBufferWithConfig[int](tt.size, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rb)
			} else {
				require.NoError(t, err)
				require.NotNil(t, rb)
				assert.Equal(t, tt.size, rb.Capacity())
			}
		})
	}
}
