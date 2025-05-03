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

func TestRingBufferBlocking(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).WithBlocking(true)
	require.NotNil(t, rb)

	n, err := rb.WriteMany([]*TestValue{{value: 1}, {value: 2}})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	done := make(chan bool)
	go func() {
		err := rb.Write(&TestValue{value: 3})
		assert.NoError(t, err)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 1, val.value)

	<-done
}

func TestRingBufferTimeout(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).
		WithBlocking(true).
		WithTimeout(100 * time.Millisecond)
	require.NotNil(t, rb)

	n, err := rb.WriteMany([]*TestValue{{value: 1}, {value: 2}})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	err = rb.Write(&TestValue{value: 3})
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	_, err = rb.GetOne()
	assert.NoError(t, err)

	_, err = rb.GetOne()
	assert.NoError(t, err)

	_, err = rb.GetOne()
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestRingBufferWriteMany(t *testing.T) {
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

	assert.Equal(t, 2, rb.Length())
}

func TestRingBufferReset(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	err := rb.Write(&TestValue{value: 1})
	assert.NoError(t, err)
	err = rb.Write(&TestValue{value: 2})
	assert.NoError(t, err)

	rb.ClearBuffer()
	assert.Equal(t, 0, rb.Length())
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

func TestRingBufferViewModification(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)

	values := []*TestValue{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
		{value: 5},
		{value: 6},
		{value: 7},
		{value: 8},
		{value: 9},
		{value: 10},
	}
	for _, v := range values {
		if err := rb.Write(v); err != nil {
			t.Fatalf("Failed to write to buffer: %v", err)
		}
	}

	part1, _, err := rb.GetManyView(5)
	if err != nil {
		t.Fatalf("Failed to get views: %v", err)
	}

	if len(part1) > 0 {
		part1[0].value = 999
		part1 = append(part1, &TestValue{value: 1000})
	}

	readValues, err := rb.GetMany(5)
	if err != nil {
		t.Fatalf("Failed to read from buffer: %v", err)
	}

	readValuesMap := make(map[int]bool)
	for _, v := range readValues {
		readValuesMap[v.value] = true
	}

	for i, v := range part1 {
		if readValuesMap[v.value] {
			t.Logf("Value at index %d was modified: got %d, want %d", i, v.value, values[i].value)
			t.Log("WARNING: Do not modify the buffer view")
		}
	}
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
	assert.ErrorAs(t, err, &context.DeadlineExceeded)
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

	_, err = rb.GetOne()
	assert.NoError(t, err)

	_, err = rb.GetOne()
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WriteMany should have completed after space was made")
	}
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
	assert.ErrorAs(t, err, &context.DeadlineExceeded)
	assert.Equal(t, 0, len(items))
}

func TestRingBufferPeekOperations(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	val, err := rb.PeekOne()
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.Nil(t, val)

	vals, err := rb.PeekMany(5)
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.Nil(t, vals)

	// Write some items
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

	// Test PeekOne
	val, err = rb.PeekOne()
	assert.NoError(t, err)
	assert.Equal(t, 1, val.value)

	// Test PeekMany
	vals, err = rb.PeekMany(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, 1, vals[0].value)
	assert.Equal(t, 2, vals[1].value)
	assert.Equal(t, 3, vals[2].value)

	// Verify items are still in buffer
	assert.Equal(t, 5, rb.Length())

	// Test PeekMany with more items than available
	vals, err = rb.PeekMany(10)
	assert.ErrorAs(t, err, &errors.ErrTooMuchDataToPeek)
	assert.Nil(t, vals)

	// Test PeekMany with zero items
	vals, err = rb.PeekMany(0)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Equal(t, 0, len(vals))

	// Test PeekMany with negative count
	vals, err = rb.PeekMany(-1)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, vals)
}

func TestRingBufferPeekViewOperations(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	part1, part2, err := rb.PeekManyView(5)
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Write some items
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

	// Test basic peek view
	part1, part2, err = rb.PeekManyView(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(part1)+len(part2))

	// Verify items are still in buffer
	assert.Equal(t, 5, rb.Length())

	// Test view modification
	if len(part1) > 0 {
		part1[0].value = 999
	}

	// Verify modification affected the buffer
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 999, val.value)

	// Test wrapping around buffer end
	// First fill the buffer
	for i := 6; i <= 10; i++ {
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Read some items to create space at the beginning
	for range 3 {
		_, err := rb.GetOne()
		assert.NoError(t, err)
	}

	// Write more items to wrap around
	for i := 11; i <= 13; i++ {
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Now peek view should return two parts
	part1, part2, err = rb.PeekManyView(5)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(part1)+len(part2))

	// Test error cases
	part1, part2, err = rb.PeekManyView(0)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.PeekManyView(-1)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.PeekManyView(20)
	assert.ErrorAs(t, err, &errors.ErrTooMuchDataToPeek)
	assert.Nil(t, part1)
	assert.Nil(t, part2)
}

func TestRingBufferGetAllView(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	part1, part2, err := rb.GetAllView()
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Write some items
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

	// Test basic get all view
	part1, part2, err = rb.GetAllView()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(part1)+len(part2))
	assert.Equal(t, 1, part1[0].value)
	assert.Equal(t, 5, part1[4].value)

	// Verify buffer is empty after getting all items
	assert.True(t, rb.IsEmpty())

	// Test view modification
	if len(part1) > 0 {
		part1[0].value = 999
	}

	// Test wrapping around buffer end
	// First fill the buffer
	for i := 6; i <= 10; i++ { // write 5 more items
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Read some items to create space at the beginning
	for range 3 { // get 3 items
		_, err := rb.GetOne()
		assert.NoError(t, err)
	}

	// Write more items to wrap around
	for i := 11; i <= 13; i++ { // write 3 more items
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Now get all view should return two parts
	part1, part2, err = rb.GetAllView()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(part1)+len(part2))

	// Verify buffer is empty after getting all items
	assert.True(t, rb.IsEmpty())
}

func TestRingBufferGetManyView(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test invalid length
	part1, part2, err := rb.GetManyView(0)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.GetManyView(-1)
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.GetManyView(11) // larger than buffer size
	assert.ErrorAs(t, err, &errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Test empty buffer
	part1, part2, err = rb.GetManyView(5)
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Write some items
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

	// Test basic get many view
	part1, part2, err = rb.GetManyView(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(part1)+len(part2))
	assert.Equal(t, 1, part1[0].value)
	assert.Equal(t, 3, part1[2].value)

	// Verify buffer still has items
	assert.Equal(t, 2, rb.Length())

	// Test view modification
	if len(part1) > 0 {
		part1[0].value = 999
	}

	// Test wrapping around buffer end
	// First fill the buffer
	for i := 6; i <= 10; i++ { // write 5 more items
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Read some items to create space at the beginning
	for range 3 { // get 3 items
		_, err := rb.GetOne()
		assert.NoError(t, err)
	}

	// Write more items to wrap around
	for i := 11; i <= 13; i++ { // write 3 more items
		err := rb.Write(&TestValue{value: i})
		assert.NoError(t, err)
	}

	// Now get many view should return two parts
	part1, part2, err = rb.GetManyView(5)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(part1)+len(part2))

	// Test blocking behavior
	rb = ringbuffer.New[*TestValue](4).WithBlocking(true)
	require.NotNil(t, rb)

	// Try to read more items than available in a goroutine
	done := make(chan bool)
	go func() {
		part1, part2, err := rb.GetManyView(4) // should block until 4 items are available
		assert.NoError(t, err)
		assert.Equal(t, 4, len(part1)+len(part2))
		done <- true
	}()

	// Verify the goroutine is blocked
	select {
	case <-done:
		t.Fatal("GetManyView should have blocked")
	case <-time.After(100 * time.Millisecond):
		// Expected - the goroutine should still be blocked
	}

	// Write items to unblock the reader
	moreItems := []*TestValue{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
	}
	n, err = rb.WriteMany(moreItems)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	// Verify the goroutine completed
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetManyView should have completed after more items were written")
	}
}

func TestRingBufferPreReadBlockHook(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Set up a hook that will try to write an item when reading from empty buffer
	hookCalled := false
	rb.WithPreReadBlockHook(func() bool {
		hookCalled = true
		err := rb.Write(&TestValue{value: 42})
		return err == nil
	})

	// Try to read from empty buffer
	val, err := rb.GetOne()
	assert.NoError(t, err)
	assert.Equal(t, 42, val.value)
	assert.True(t, hookCalled)

	// Test hook that returns false
	hookCalled = false
	rb.WithPreReadBlockHook(func() bool {
		hookCalled = true
		return false
	})

	// Try to read from empty buffer again
	_, err = rb.GetOne()
	assert.ErrorAs(t, err, &errors.ErrIsEmpty)
	assert.True(t, hookCalled)
}

func TestRingBufferGetManyViewTimeout(t *testing.T) {
	rb := ringbuffer.New[*TestValue](2).WithTimeout(100 * time.Millisecond)
	require.NotNil(t, rb)

	// Try to get more items than available with timeout
	part1, part2, err := rb.GetManyView(5)
	assert.ErrorAs(t, err, &context.DeadlineExceeded)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Write one item and try to get more than available
	err = rb.Write(&TestValue{value: 1})
	assert.NoError(t, err)

	part1, part2, err = rb.GetManyView(2)
	assert.ErrorAs(t, err, &context.DeadlineExceeded)
	assert.Nil(t, part1)
	assert.Nil(t, part2)
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
