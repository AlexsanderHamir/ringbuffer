package test

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
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

	err := rb.Write(&TestValue{value: 1})
	assert.NoError(t, err)
	err = rb.Write(&TestValue{value: 2})
	assert.NoError(t, err)

	rb.ClearBuffer()
	assert.Equal(t, 0, rb.Length(false))
	assert.True(t, rb.IsEmpty())
}

func TestRingBufferEdgeCases(t *testing.T) {
	rb := ringbuffer.New[*TestValue](0)
	assert.Nil(t, rb)

	rb = ringbuffer.New[*TestValue](-1)
	assert.Nil(t, rb)

	rb = ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)
}
