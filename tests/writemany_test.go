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

func TestRingBufferWriteManyBasic(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	items := []*TestValue{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
		{value: 5},
	}
	n, err := rb.WriteMany(items)
	assert.NoError(t, err)
	assert.Equal(t, len(items), n)
}

func TestRingBufferWriteManyOverflow(t *testing.T) {
	rb := ringbuffer.New[*TestValue](4)
	require.NotNil(t, rb)

	items := []*TestValue{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
		{value: 5},
	}

	n, err := rb.WriteMany(items)
	assert.ErrorIs(t, err, errors.ErrIsFull)
	assert.Equal(t, 0, n)
}

func TestRingBufferWriteManyTimeout(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).WithTimeout(100 * time.Millisecond)
	require.NotNil(t, rb)
	defer rb.Close()

	items := []*TestValue{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
		{value: 5},
	}
	n, err := rb.WriteMany(items)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 0, n)
}

func TestRingBufferWriteManyBlocking(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).WithBlocking(true)
	require.NotNil(t, rb)
	defer rb.Close()

	// First write to fill the buffer
	items := []*TestValue{
		{value: 1},
		{value: 2},
	}
	n, err := rb.WriteMany(items)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// Try to write more items in a goroutine
	done := make(chan bool)
	go func() {
		moreItems := []*TestValue{
			{value: 3},
			{value: 4},
		}
		n, err := rb.WriteMany(moreItems)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		done <- true
	}()

	// Verify the goroutine is blocked
	select {
	case <-done:
		t.Fatal("WriteMany should have blocked")
	case <-time.After(100 * time.Millisecond):
		// Expected - the goroutine should still be blocked
	}

	_, err = rb.GetMany(2)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WriteMany should have completed after space was made")
	}
}
