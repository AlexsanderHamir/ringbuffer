package test

import (
	"context"
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCases(t *testing.T) {
	t.Run("Buffer State Errors", func(t *testing.T) {
		rb := ringbuffer.New[int](5)
		require.NotNil(t, rb)

		// Test ErrIsEmpty
		_, err := rb.GetOne()
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		_, err = rb.PeekOne()
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		// Fill buffer
		for i := range 5 {
			err := rb.Write(i)
			require.NoError(t, err)
		}

		// Test ErrIsFull
		err = rb.Write(6)
		assert.ErrorIs(t, err, errors.ErrIsFull)

		// Test ErrIsFull with WriteMany
		_, err = rb.WriteMany([]int{1, 2, 3})
		assert.ErrorIs(t, err, errors.ErrIsFull)
	})

	t.Run("Timeout Errors", func(t *testing.T) {
		rb := ringbuffer.New[int](1)
		require.NotNil(t, rb)

		// Set very short timeouts
		rb.WithTimeout(1 * time.Millisecond)

		// Fill buffer
		err := rb.Write(1)
		require.NoError(t, err)

		// Test write timeout
		err = rb.Write(2)
		assert.ErrorIs(t, err, context.DeadlineExceeded)

		// Empty the buffer
		_, err = rb.GetOne()
		require.NoError(t, err)

		// Test read timeout
		_, err = rb.GetOne()
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("View Operation Errors", func(t *testing.T) {
		rb := ringbuffer.New[int](5)
		require.NotNil(t, rb)

		// Test view on empty buffer
		_, _, err := rb.GetNView(1)
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		// Fill buffer partially
		for i := range 3 {
			err := rb.Write(i)
			require.NoError(t, err)
		}

		// Test view with invalid length
		_, _, err = rb.GetNView(0)
		assert.ErrorIs(t, err, errors.ErrInvalidLength)

		_, _, err = rb.GetNView(-1)
		assert.ErrorIs(t, err, errors.ErrInvalidLength)

		_, _, err = rb.GetNView(6)
		assert.ErrorIs(t, err, errors.ErrInvalidLength)

		// Test GetAllView on empty buffer
		rb.ClearBuffer()
		_, _, err = rb.GetAllView()
		assert.ErrorIs(t, err, errors.ErrIsEmpty)
	})

	t.Run("Hook Error Cases", func(t *testing.T) {
		rb := ringbuffer.New[*int](1)
		require.NotNil(t, rb)

		// Test hook errors
		rb.WithPreReadBlockHook(func() (*int, bool, bool) {
			return nil, false, false // Simulate hook failure
		})

		obj, err := rb.GetOne()
		assert.ErrorIs(t, err, errors.ErrIsEmpty)
		assert.Nil(t, obj)
		rb.WithPreWriteBlockHook(func() bool {
			return false
		})

		int1 := 1
		rb.Write(&int1) // fill buffer

		err = rb.Write(&int1)
		assert.ErrorIs(t, err, errors.ErrIsFull)
	})

	t.Run("Edge Cases", func(t *testing.T) {
		// Test with minimum valid size
		rb := ringbuffer.New[int](1)
		require.NotNil(t, rb)

		// Test write/read on minimum size buffer
		err := rb.Write(1)
		require.NoError(t, err)

		val, err := rb.GetOne()
		require.NoError(t, err)
		assert.Equal(t, 1, val)

		// Test with maximum practical size
		rb = ringbuffer.New[int](1000000)
		require.NotNil(t, rb)

		// Test write/read on large buffer
		for i := range 1000 {
			err := rb.Write(i)
			require.NoError(t, err)
		}

		for i := range 1000 {
			val, err := rb.GetOne()
			require.NoError(t, err)
			assert.Equal(t, i, val)
		}
	})

	t.Run("Peek Operation Errors", func(t *testing.T) {
		rb := ringbuffer.New[int](5)
		require.NotNil(t, rb)

		// Test peek on empty buffer
		_, err := rb.PeekOne()
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		_, err = rb.PeekN(1)
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		// Fill buffer partially
		for i := range 3 {
			err := rb.Write(i)
			require.NoError(t, err)
		}

		// Test peek with invalid length
		_, err = rb.PeekN(0)
		assert.ErrorIs(t, err, errors.ErrInvalidLength)

		_, err = rb.PeekN(-1)
		assert.ErrorIs(t, err, errors.ErrInvalidLength)

		// Test peek with too many items
		_, err = rb.PeekN(4)
		assert.ErrorIs(t, err, errors.ErrTooMuchDataToPeek)
	})

	t.Run("WriteMany Edge Cases", func(t *testing.T) {
		rb := ringbuffer.New[int](5)
		require.NotNil(t, rb)

		// Test writing nil slice
		n, err := rb.WriteMany(nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)

		// Test writing empty slice
		n, err = rb.WriteMany([]int{})
		assert.NoError(t, err)
		assert.Equal(t, 0, n)

		// Test writing more items than buffer size
		items := make([]int, 6)
		n, err = rb.WriteMany(items)
		assert.ErrorIs(t, err, errors.ErrIsFull)
		assert.Equal(t, 0, n)
	})

	t.Run("GetMany Edge Cases", func(t *testing.T) {
		rb := ringbuffer.New[int](5)
		require.NotNil(t, rb)

		// Test getting more items than buffer size
		_, err := rb.GetN(6)
		assert.ErrorIs(t, err, errors.ErrIsEmpty)

		// Fill buffer
		for i := range 5 {
			err := rb.Write(i)
			require.NoError(t, err)
		}

		// Test getting all items
		items, err := rb.GetN(5)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(items))

		// Test getting more items than available
		_, err = rb.GetN(1)
		assert.ErrorIs(t, err, errors.ErrIsEmpty)
	})
}
