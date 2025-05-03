package test

import (
	"context"
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBufferGetManyBasic(t *testing.T) {
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

	vals, err := rb.GetMany(3)
	assert.NoError(t, err)
	assert.Equal(t, 1, vals[0].value)
	assert.Equal(t, 2, vals[1].value)
	assert.Equal(t, 3, vals[2].value)

	assert.Equal(t, 2, rb.Length(false))
}

func TestRingBufferGetManyBlocking(t *testing.T) {
	rb := ringbuffer.New[*TestValue](4).WithBlocking(true)
	require.NotNil(t, rb)
	defer rb.Close()

	// First write some items
	items := []*TestValue{
		{value: 1},
		{value: 2},
	}
	n, err := rb.WriteMany(items)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// Try to read more items than available in a goroutine
	done := make(chan bool)
	go func() {
		vals, err := rb.GetMany(4) // should block until 4 items are available
		assert.NoError(t, err)
		assert.Equal(t, 4, len(vals))
		done <- true
	}()

	// Verify the goroutine is blocked
	select {
	case <-done:
		t.Fatal("GetMany should have blocked")
	case <-time.After(100 * time.Millisecond):
		// Expected - the goroutine should still be blocked
	}

	// Write more items to unblock the reader
	moreItems := []*TestValue{
		{value: 3},
		{value: 4},
	}
	n, err = rb.WriteMany(moreItems)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// Verify the goroutine completed
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetMany should have completed after more items were written")
	}
}

func TestRingBufferGetManyTimeout(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).WithTimeout(100 * time.Millisecond)
	require.NotNil(t, rb)

	items, err := rb.GetMany(5)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 0, len(items))
}
