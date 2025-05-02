package tests

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMany(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		items     []int
		wantN     int
		wantError bool
	}{
		{
			name:      "write within capacity",
			capacity:  5,
			items:     []int{1, 2, 3},
			wantN:     3,
			wantError: false,
		},
		{
			name:      "write empty slice",
			capacity:  5,
			items:     []int{},
			wantN:     0,
			wantError: false,
		},
		{
			name:      "write more than capacity",
			capacity:  2,
			items:     []int{1, 2, 3},
			wantN:     0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := ringbuffer.NewRingBuffer[int](tt.capacity)
			require.NotNil(t, rb)

			n, err := rb.WriteMany(tt.items)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, 0, n)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantN, n)
				assert.Equal(t, tt.wantN, rb.Length(false))
			}
		})
	}
}

func TestGetN(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		write     []int
		readN     int
		wantItems []int
		wantError bool
	}{
		{
			name:      "read within available",
			capacity:  5,
			write:     []int{1, 2, 3},
			readN:     2,
			wantItems: []int{1, 2},
			wantError: false,
		},
		{
			name:      "read all available",
			capacity:  5,
			write:     []int{1, 2, 3},
			readN:     3,
			wantItems: []int{1, 2, 3},
			wantError: false,
		},
		{
			name:      "read more than available",
			capacity:  5,
			write:     []int{1, 2},
			readN:     3,
			wantItems: []int{1, 2},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := ringbuffer.NewRingBuffer[int](tt.capacity)
			require.NotNil(t, rb)

			_, err := rb.WriteMany(tt.write)
			require.NoError(t, err)

			items, err := rb.GetN(tt.readN)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantItems, items)
			}
		})
	}
}

func TestTryOperations(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](3)
	require.NotNil(t, rb)

	t.Run("try write and read", func(t *testing.T) {
		// Try write to empty buffer
		assert.True(t, rb.TryWrite(1))
		assert.True(t, rb.TryWrite(2))
		assert.True(t, rb.TryWrite(3))
		assert.False(t, rb.TryWrite(4)) // Buffer is full

		// Try read from full buffer
		val, ok := rb.TryRead()
		assert.True(t, ok)
		assert.Equal(t, 1, val)

		val, ok = rb.TryRead()
		assert.True(t, ok)
		assert.Equal(t, 2, val)

		val, ok = rb.TryRead()
		assert.True(t, ok)
		assert.Equal(t, 3, val)

		// Try read from empty buffer
		_, ok = rb.TryRead()
		assert.False(t, ok)
	})
}

func TestGetAllView(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		write     []int
		wantPart1 []int
		wantPart2 []int
		wantError bool
	}{
		{
			name:      "empty buffer",
			capacity:  5,
			write:     []int{},
			wantPart1: nil,
			wantPart2: nil,
			wantError: true,
		},
		{
			name:      "continuous data",
			capacity:  5,
			write:     []int{1, 2, 3},
			wantPart1: []int{1, 2, 3},
			wantPart2: nil,
			wantError: false,
		},
		{
			name:      "wrapped data",
			capacity:  3,
			write:     []int{1, 2, 3, 4, 5},
			wantPart1: []int{3},
			wantPart2: []int{4, 5},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := ringbuffer.NewRingBuffer[int](tt.capacity)
			require.NotNil(t, rb)

			if tt.name == "wrapped data" {
				rb.WithOverwrite(true)
				rb.WithBlocking(true)
			}

			for _, item := range tt.write {
				err := rb.Write(item)
				require.NoError(t, err)
			}

			part1, part2, err := rb.GetAllView()
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, part1)
				assert.Nil(t, part2)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPart1, part1)
				assert.Equal(t, tt.wantPart2, part2)
			}
		})
	}
}
