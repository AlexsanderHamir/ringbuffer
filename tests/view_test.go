package test

import (
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestRingBufferGetAllView(t *testing.T) {
	rb := ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test empty buffer
	part1, part2, err := rb.GetAllView()
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
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.GetManyView(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	part1, part2, err = rb.GetManyView(11) // larger than buffer size
	assert.ErrorIs(t, err, errors.ErrInvalidLength)
	assert.Nil(t, part1)
	assert.Nil(t, part2)

	// Test empty buffer
	part1, part2, err = rb.GetManyView(5)
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

	// Test basic get many view
	part1, part2, err = rb.GetManyView(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(part1)+len(part2))
	assert.Equal(t, 1, part1[0].value)
	assert.Equal(t, 3, part1[2].value)

	// Verify buffer still has items
	assert.Equal(t, 2, rb.Length(false))

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
}
