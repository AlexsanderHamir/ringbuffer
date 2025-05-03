package test

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestValue struct {
	value int
}

func TestRingBufferBasicOperations(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	err := rb.Write(&TestValue{value: 42})
	assert.NoError(t, err)

	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 42, val.value)

	assert.Equal(t, 10, rb.Capacity())

	assert.True(t, rb.IsEmpty())
	assert.False(t, rb.IsFull())
}

func TestRingBufferReset(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Write some items
	err := rb.Write(&TestValue{value: 1})
	assert.NoError(t, err)
	err = rb.Write(&TestValue{value: 2})
	assert.NoError(t, err)

	// Reset the buffer
	rb.Reset()

	// Verify buffer is empty
	assert.Equal(t, 0, rb.Length(false))
	assert.True(t, rb.IsEmpty())
	assert.False(t, rb.IsFull())

	// Verify we can write after reset
	err = rb.Write(&TestValue{value: 3})
	assert.NoError(t, err)
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 3, val.value)
}

func TestRingBufferFlush(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Write some items
	err := rb.Write(&TestValue{value: 1})
	assert.NoError(t, err)
	err = rb.Write(&TestValue{value: 2})
	assert.NoError(t, err)

	// Fill the buffer to trigger ErrIsFull
	for i := range 10 {
		err = rb.Write(&TestValue{value: i})
		if err != nil {
			break
		}
	}

	assert.ErrorIs(t, err, errors.ErrIsFull)

	// Flush the buffer
	rb.Flush()

	// Verify buffer is empty
	assert.Equal(t, 0, rb.Length(false))
	assert.True(t, rb.IsEmpty())
	assert.False(t, rb.IsFull())

	// Verify we can write after flush
	err = rb.Write(&TestValue{value: 3})
	assert.NoError(t, err)
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 3, val.value)
}

func TestRingBufferEdgeCases(t *testing.T) {
	rb := ringbuffer.New[*TestValue](0)
	assert.Nil(t, rb)

	rb = ringbuffer.New[*TestValue](-1)
	assert.Nil(t, rb)

	rb = ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)
}
