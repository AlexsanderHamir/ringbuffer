package test

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBufferPeekOperations(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	val, err := rb.PeekOne()
	assert.ErrorIs(t, err, errors.ErrIsEmpty)
	assert.Nil(t, val)

	vals, err := rb.PeekMany(5)
	assert.ErrorIs(t, err, errors.ErrIsEmpty)
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
	assert.Equal(t, 5, rb.Length(false))

	// Test PeekMany with more items than available
	vals, err = rb.PeekMany(10)
	assert.ErrorIs(t, err, errors.ErrTooMuchDataToPeek)
	assert.Nil(t, vals)

	// Test PeekMany with zero items
	vals, err = rb.PeekMany(0)
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Equal(t, 0, len(vals))

	// Test PeekMany with negative count
	vals, err = rb.PeekMany(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, vals)
}

func TestRingBufferPeekViewOperations(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	part1, part2, err := rb.PeekManyView(5)
	assert.ErrorIs(t, err, errors.ErrIsEmpty)
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
	assert.Equal(t, 5, rb.Length(false))

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
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.PeekManyView(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.PeekManyView(20)
	assert.ErrorIs(t, err, errors.ErrTooMuchDataToPeek)
	assert.Nil(t, part1)
	assert.Nil(t, part2)
}
