package ringbuffer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/AlexsanderHamir/ringbuffer/errors"
)

// setErr sets the error state of the ring buffer
func (r *RingBuffer[T]) setErr(err error, locked bool) error {
	if !locked {
		r.mu.Lock()
		defer r.mu.Unlock()
	}

	if r.err != nil && r.err != io.EOF {
		return r.err
	}

	switch err {
	// Internal errors are temporary
	case nil, errors.ErrIsEmpty, errors.ErrIsFull, errors.ErrAcquireLock, errors.ErrTooMuchDataToWrite, errors.ErrIsNotEmpty, context.DeadlineExceeded:
		return err
	default:
		r.err = err
		if r.block {
			r.readCond.Broadcast()
			r.writeCond.Broadcast()
		}
	}
	return err
}

// readErr checks for errors in the ring buffer
func (r *RingBuffer[T]) readErr(locked bool, log bool, location string) error {
	if !locked {
		r.mu.Lock()
		defer r.mu.Unlock()
	}

	if r.err != nil {
		if r.err == io.EOF {
			if r.w == r.r && !r.isFull {
				if log {
					fmt.Println("readErr EOF: ", location)
				}
				return io.EOF
			}
			return nil
		}
		return r.err
	}
	return nil
}

// waitRead waits for a read event
// Returns true if a read may have happened.
// Returns false if waited longer than rTimeout.
// Must be called when locked and returns locked.
func (r *RingBuffer[T]) waitRead() (ok bool) {
	r.blockedWriters++

	defer func() { r.blockedWriters-- }()

	if r.rTimeout <= 0 {
		r.readCond.Wait()
		return true
	}

	start := time.Now()
	defer time.AfterFunc(r.rTimeout, r.readCond.Broadcast).Stop()

	r.readCond.Wait()
	if time.Since(start) >= r.rTimeout {
		r.setErr(context.DeadlineExceeded, true)
		return false
	}

	return true
}

// waitWrite waits for a write event
// Returns true if a write may have happened.
// Returns false if waited longer than wTimeout.
// Must be called when locked and returns locked.
func (r *RingBuffer[T]) waitWrite() (ok bool) {
	r.blockedReaders++

	defer func() {
		r.blockedReaders--
	}()

	if r.wTimeout <= 0 {
		r.writeCond.Wait()
		return true
	}

	start := time.Now()

	defer time.AfterFunc(r.wTimeout, r.writeCond.Broadcast).Stop()

	r.writeCond.Wait()
	if time.Since(start) >= r.wTimeout {
		r.setErr(context.DeadlineExceeded, true)
		return false
	}

	return true
}
