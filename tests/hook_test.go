package test

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBufferPreReadBlockHook(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Set up a hook that will try to write an item when reading from empty buffer
	hookCalled := false
	rb.WithPreReadBlockHook(func() (*TestValue, bool, bool) {
		hookCalled = true
		err := rb.Write(&TestValue{value: 42})
		return nil, err == nil, true
	})

	// Try to read from empty buffer
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 42, val.value)
	assert.True(t, hookCalled)

	// Test hook that returns false
	hookCalled = false
	rb.WithPreReadBlockHook(func() (*TestValue, bool, bool) {
		hookCalled = true
		return nil, false, false
	})

	// Try to read from empty buffer again
	_, err = rb.GetOne()
	assert.ErrorIs(t, err, errors.ErrIsEmpty)
	assert.True(t, hookCalled)
}

func TestRingBufferPreWriteBlockHook(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2)
	require.NotNil(t, rb)

	// Test hook that successfully handles the full buffer
	// change the hook to free space in the buffer
	hookCalled := false
	rb.WithPreWriteBlockHook(func() bool {
		hookCalled = true
		_, err := rb.GetOne()
		return err == nil
	})

	// Fill the buffer
	n, err := rb.WriteMany([]*TestValue{{value: 1}, {value: 2}})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// Try to write when buffer is full
	err = rb.Write(&TestValue{value: 3})
	assert.NoError(t, err)
	assert.True(t, hookCalled)
}
