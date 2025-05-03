package test

import (
	"io"
	"testing"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBufferInvalidOperations(t *testing.T) {
	// Test invalid buffer creation
	rb := ringbuffer.New[*TestValue](0)
	assert.Nil(t, rb, "Buffer with size 0 should be nil")

	rb = ringbuffer.New[*TestValue](-1)
	assert.Nil(t, rb, "Buffer with negative size should be nil")

	// Test invalid operations on nil buffer
	rb = nil
	err := rb.Write(&TestValue{value: 1})
	assert.ErrorIs(t, err, errors.ErrNilBuffer)

	_, err = rb.GetOne()
	assert.ErrorIs(t, err, errors.ErrNilBuffer)

	_, err = rb.GetMany(1)
	assert.ErrorIs(t, err, errors.ErrNilBuffer)

	_, err = rb.PeekOne()
	assert.ErrorIs(t, err, errors.ErrNilBuffer)

	_, err = rb.PeekMany(1)
	assert.ErrorIs(t, err, errors.ErrNilBuffer)

	// Test invalid operations on valid buffer
	rb = ringbuffer.New[*TestValue](10)
	require.NotNil(t, rb)

	// Test invalid GetMany/PeekMany lengths
	_, err = rb.GetMany(0)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "GetMany with length 0 should return ErrInvalidLength")

	_, err = rb.GetMany(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "GetMany with negative length should return ErrInvalidLength")

	_, err = rb.GetMany(11)
	assert.ErrorIs(t, err, errors.ErrIsEmpty)

	_, err = rb.PeekMany(0)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "PeekMany with length 0 should return ErrInvalidLength")

	_, err = rb.PeekMany(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "PeekMany with negative length should return ErrInvalidLength")

	// Test operations on closed buffer
	err = rb.Close()
	assert.NoError(t, err, "Close should not return error")

	err = rb.Write(&TestValue{value: 1})
	assert.ErrorIs(t, err, io.EOF, "Write on closed buffer should return EOF")

	_, err = rb.GetOne()
	assert.ErrorIs(t, err, io.EOF, "GetOne on closed buffer should return EOF")

	_, err = rb.GetMany(1)
	assert.ErrorIs(t, err, io.EOF, "GetMany on closed buffer should return EOF")

	_, err = rb.PeekOne()
	assert.ErrorIs(t, err, io.EOF, "PeekOne on closed buffer should return EOF")

	_, err = rb.PeekMany(1)
	assert.ErrorIs(t, err, io.EOF, "PeekMany on closed buffer should return EOF")

	// Test invalid view operations
	_, _, err = rb.GetManyView(0)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "GetManyView with length 0 should return ErrInvalidLength")

	_, _, err = rb.GetManyView(-1)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "GetManyView with negative length should return ErrInvalidLength")

	_, _, err = rb.GetManyView(11)
	assert.ErrorIs(t, err, errors.ErrInvalidLength, "GetManyView with length > buffer size should return ErrInvalidLength")

	// Test invalid WriteMany operations
	_, err = rb.WriteMany(nil)
	assert.NoError(t, err, "WriteMany with nil slice should not return error")
}
