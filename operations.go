package ringbuffer

import (
	"context"
	"github.com/AlexsanderHamir/ringbuffer/errors"
)

// Write writes a single item to the buffer.
// Behavior:
// - Blocks if buffer is full and in blocking mode
// - Returns ErrIsFull if buffer is full and not blocking
// - Returns context.DeadlineExceeded if timeout occurs
// - Signals waiting readers when data is written
func (r *RingBuffer[T]) Write(item T) error { // tested
	if r == nil {
		return errors.ErrNilBuffer
	}

	r.mu.Lock()
	defer func() {
		if r.block && r.blockedReaders > 0 {
			r.writeCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "Write"); err != nil {
		return err
	}

	wblockAttempts := 1
	for r.isFull {
		if r.preWriteBlockHook != nil {
			r.mu.Unlock()
			tryAgain := r.preWriteBlockHook()
			r.mu.Lock()
			if tryAgain && wblockAttempts > 0 {
				wblockAttempts--
				continue
			}
		}

		if !r.block {
			return errors.ErrIsFull
		}

		if !r.waitRead() {
			return context.DeadlineExceeded
		}
	}

	r.buf[r.w] = item
	r.w = (r.w + 1) % r.size
	if r.w == r.r {
		r.isFull = true
	}

	return nil
}

// WriteMany writes multiple items to the buffer.
// Behavior:
// - Writes all items or none
// - Returns ErrIsFull if buffer doesn't have enough space and not blocking
// - Blocks until all items can be written or timeout occurs
// - Returns number of items written and any error
// - Handles wrapping around the buffer end
func (r *RingBuffer[T]) WriteMany(items []T) (n int, err error) { // tested
	if len(items) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer func() {
		if r.block && n > 0 {
			r.writeCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "WriteMany"); err != nil {
		return 0, err
	}

	// Calculate available free space, not total items.
	availableSpace := r.availableSpace()
	wblockAttempts := 1
	// If we don't have enough free space
	for len(items) > availableSpace {
		if r.preWriteBlockHook != nil {
			r.mu.Unlock()
			tryAgain := r.preWriteBlockHook()
			r.mu.Lock()
			if tryAgain && wblockAttempts > 0 {
				wblockAttempts--
				continue
			}
		}

		if !r.block {
			return 0, errors.ErrIsFull
		}

		if !r.waitRead() {
			return 0, context.DeadlineExceeded
		}

		// Recalculate available space after being woken up
		availableSpace = r.availableSpace()
	}

	// Write all items
	if r.w+len(items) <= r.size {
		// Can write in one go
		copy(r.buf[r.w:], items)
	} else {
		// Need to wrap around
		firstPart := r.size - r.w
		copy(r.buf[r.w:], items[:firstPart])
		copy(r.buf[0:], items[firstPart:])
	}
	r.w = (r.w + len(items)) % r.size
	r.isFull = r.w == r.r
	n = len(items)

	return n, nil
}

// GetOne returns a single item from the buffer.
// Behavior:
// - Blocks if buffer is empty and in blocking mode
// - Returns ErrIsEmpty if buffer is empty and not blocking
// - Returns context.DeadlineExceeded if timeout occurs
// - Signals waiting writers when data is read
func (r *RingBuffer[T]) GetOne() (item T, err error) { // tested
	if r == nil {
		return item, errors.ErrNilBuffer
	}

	r.mu.Lock()
	defer func() {
		if r.block && r.blockedWriters > 0 {
			r.readCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "GetOne_First"); err != nil {
		return item, err
	}

	rblockAttempts := 1
	for r.w == r.r && !r.isFull {
		if r.preReadBlockHook != nil {
			r.mu.Unlock()
			obj, tryAgain, success := r.preReadBlockHook()
			r.mu.Lock()
			if tryAgain && rblockAttempts > 0 {
				rblockAttempts--
				continue
			}

			if success {
				return obj, nil
			}
		}

		if !r.block {
			return item, errors.ErrIsEmpty
		}

		if !r.waitWrite() {
			return item, context.DeadlineExceeded
		}

		if err := r.readErr(true, false, "GetOne_InnerBlock"); err != nil {
			return item, err
		}
	}

	item = r.buf[r.r]
	r.r = (r.r + 1) % r.size
	r.isFull = false

	return item, r.readErr(true, false, "GetOne_Second")
}

// GetMany returns n items from the buffer.
// Behavior:
// - Gets all n items or blocks until it can
// - Returns ErrIsEmpty if buffer is empty and not blocking
// - Returns context.DeadlineExceeded if timeout occurs
// - Handles wrapping around the buffer end
func (r *RingBuffer[T]) GetN(n int) (items []T, err error) { // tested
	if r == nil {
		return nil, errors.ErrNilBuffer
	}

	if n <= 0 {
		return nil, errors.ErrInvalidLength
	}

	r.mu.Lock()
	defer func() {
		if r.block && r.blockedWriters > 0 {
			r.readCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "GetN"); err != nil {
		return nil, err
	}

	// Calculate how many items we can read
	availableItems := r.Length(true)

	// Keep waiting until we can read all n items
	for n > availableItems {
		if !r.block {
			return nil, errors.ErrIsEmpty
		}

		if !r.waitWrite() {
			return nil, context.DeadlineExceeded
		}

		if err := r.readErr(true, false, "GetN"); err != nil {
			return nil, err
		}

		// Recalculate available items after being woken up
		availableItems = r.Length(true)
	}

	// Create result slice and copy data
	items = make([]T, n)
	if r.w > r.r || n <= r.size-r.r {
		// Can read in one go
		copy(items, r.buf[r.r:r.r+n])
	} else {
		// Need to wrap around
		firstPart := r.size - r.r
		copy(items, r.buf[r.r:r.size])
		copy(items[firstPart:], r.buf[0:n-firstPart])
	}

	r.r = (r.r + n) % r.size
	r.isFull = false

	return items, r.readErr(true, false, "GetN")
}

// PeekOne returns the next item without removing it from the buffer
func (r *RingBuffer[T]) PeekOne() (item T, err error) { // tested
	if r == nil {
		return item, errors.ErrNilBuffer
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.readErr(true, false, "PeekOne"); err != nil {
		return item, err
	}

	if r.w == r.r && !r.isFull {
		return item, errors.ErrIsEmpty
	}

	return r.buf[r.r], nil
}

// PeekMany returns exactly n items without removing them from the buffer.
// Returns ErrIsEmpty if there aren't enough items available.
func (r *RingBuffer[T]) PeekN(n int) (items []T, err error) { // tested
	if n <= 0 {
		return nil, errors.ErrInvalidLength
	}

	if r == nil {
		return nil, errors.ErrNilBuffer
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.readErr(true, false, "PeekN"); err != nil {
		return nil, err
	}

	if r.w == r.r && !r.isFull {
		return nil, errors.ErrIsEmpty
	}

	availableItems := r.Length(true) // available objects not space

	if n > availableItems {
		return nil, errors.ErrTooMuchDataToPeek
	}

	items = make([]T, n)
	if r.w > r.r {
		copy(items, r.buf[r.r:r.r+n])
	} else {
		c1 := r.size - r.r
		if c1 >= n {
			copy(items, r.buf[r.r:r.r+n])
		} else {
			copy(items, r.buf[r.r:r.size])
			copy(items[c1:], r.buf[0:n-c1])
		}
	}

	return items, nil
}

// PeekManyView returns a view of exactly n items from the buffer without removing them.
// The view is not a copy, but a reference to the buffer.
// The view is valid until the buffer is modified.
// If the view is modified, the buffer will be modified.
// Make sure to get the items out of the slice before the buffer is modified.
// This is more efficient than PeekN, but less safe, depending on your use case.
// Returns ErrIsEmpty if there aren't exactly n items available.
func (r *RingBuffer[T]) PeekNView(n int) (part1, part2 []T, err error) { // tested
	if n <= 0 {
		return nil, nil, errors.ErrInvalidLength
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.readErr(true, false, "PeekManyView"); err != nil {
		return nil, nil, err
	}

	if r.w == r.r && !r.isFull {
		return nil, nil, errors.ErrIsEmpty
	}

	availableItems := r.Length(true)

	if n > availableItems {
		return nil, nil, errors.ErrTooMuchDataToPeek
	}

	if r.w > r.r || n <= r.size-r.r {
		part1 = r.buf[r.r : r.r+n]
	} else {
		part1 = r.buf[r.r:r.size]
		part2 = r.buf[0 : n-len(part1)]
	}

	return part1, part2, nil
}

// GetAllView returns a view of all items in the buffer.
// The view is not a copy, but a reference to the buffer.
// The view is valid until the buffer is modified.
// If the view is modified, the buffer will be modified.
// Make sure to get the items out of the slice before the buffer is modified.
// This is more efficient than GetAll, but less safe, depending on your use case.
// Returns ErrIsEmpty if the buffer is empty.
func (r *RingBuffer[T]) GetAllView() (part1, part2 []T, err error) { // tested
	r.mu.Lock()
	defer func() {
		if r.block && r.blockedWriters > 0 {
			r.readCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "GetAllView"); err != nil {
		return nil, nil, err
	}

	if r.w == r.r && !r.isFull {
		return nil, nil, errors.ErrIsEmpty
	}

	if r.w > r.r {
		part1 = r.buf[r.r:r.w]
	} else {
		part1 = r.buf[r.r:r.size]
		part2 = r.buf[0:r.w]
	}

	r.r = r.w
	r.isFull = false

	return part1, part2, r.readErr(true, false, "GetAllView")
}

// GetManyView returns a view of exactly n items from the buffer.
// The view is not a copy, but a reference to the buffer.
// The view is valid until the buffer is modified.
// If the view is modified, the buffer will be modified.
// Make sure to get the items out of the slice before the buffer is modified.
// This is more efficient than GetMany, but less safe, depending on your use case.
// Returns:
// - ErrInvalidLength if n <= 0 or n > buffer size
// - ErrIsEmpty if buffer is empty and not blocking
// - context.DeadlineExceeded if timeout occurs
func (r *RingBuffer[T]) GetNView(n int) (part1, part2 []T, err error) { // tested
	if n <= 0 {
		return nil, nil, errors.ErrInvalidLength
	}

	// otherwise it will block forever
	if n > r.size {
		return nil, nil, errors.ErrInvalidLength
	}

	r.mu.Lock()
	defer func() {
		if r.block && r.blockedWriters > 0 {
			r.readCond.Signal()
		}
		r.mu.Unlock()
	}()

	if err := r.readErr(true, false, "GetNView"); err != nil {
		return nil, nil, err
	}

	// Calculate how many items we can read
	available := r.Length(true)

	for available < n || r.w == r.r && !r.isFull {
		if !r.block {
			return nil, nil, errors.ErrIsEmpty
		}

		if !r.waitWrite() {
			return nil, nil, context.DeadlineExceeded
		}

		if err := r.readErr(true, false, "GetNView"); err != nil {
			return nil, nil, err
		}

		// Recalculate available items after being woken up
		available = r.Length(true)
	}

	if r.w > r.r || n <= r.size-r.r {
		part1 = r.buf[r.r : r.r+n]
	} else {
		part1 = r.buf[r.r:r.size]
		part2 = r.buf[0 : n-len(part1)]
	}

	r.r = (r.r + n) % r.size
	r.isFull = false

	return part1, part2, r.readErr(true, false, "GetNView")
}

// availableSpace returns the number of free slots in the buffer.
func (r *RingBuffer[T]) availableSpace() int {
	if r.isFull {
		return 0
	}
	if r.w >= r.r {
		return r.size - r.w + r.r
	}
	return r.r - r.w
}

// wake up one reader
func (r *RingBuffer[T]) WakeUpOneReader() {
	if r.block && r.blockedReaders > 0 {
		r.writeCond.Signal()
	}
}

// wake up one writer
func (r *RingBuffer[T]) WakeUpOneWriter() {
	if r.block && r.blockedWriters > 0 {
		r.readCond.Signal()
	}
}
