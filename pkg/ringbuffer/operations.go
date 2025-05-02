package ringbuffer

import (
	"time"
)

// Write writes a single item to the buffer.
// If the buffer is full and blocking is enabled, it will block until space is available.
// If the buffer is full and blocking is disabled, it returns ErrIsFull.
func (r *RingBuffer[T]) Write(item T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("Write operation started")

	if r.paused {
		r.logVerbose("Write operation skipped - buffer is paused")
		return nil
	}

	for r.isFull {
		r.logVerbose("Buffer is full, handling full state")
		if err := r.handleBufferFull(); err != nil {
			r.logVerbose("Write operation failed: %v", err)
			return err
		}
	}

	r.buf[r.w] = item
	r.w = (r.w + 1) % r.size
	r.isFull = r.w == r.r

	r.logVerbose("Item written successfully, new write position: %d, isFull: %v", r.w, r.isFull)
	r.notifyWriteHandlers(item)

	if r.block && r.blockedReaders.Load() > 0 {
		r.logVerbose("Signaling blocked readers")
		r.writeCond.Signal()
	}

	r.logVerbose("Write operation completed")
	return nil
}

// WriteMany writes multiple items to the buffer.
// Returns the number of items written and any error encountered.
func (r *RingBuffer[T]) WriteMany(items []T) (n int, err error) {
	if len(items) == 0 {
		return 0, nil
	}

	if len(items) > r.size {
		return 0, ErrTooMuchDataToWrite
	}

	for _, item := range items {
		if err := r.Write(item); err != nil {
			return n, err
		}
		n++
	}

	return n, nil
}

// GetOne reads a single item from the buffer.
// If the buffer is empty and blocking is enabled, it will block until data is available.
// If the buffer is empty and blocking is disabled, it returns ErrIsEmpty.
func (r *RingBuffer[T]) GetOne() (item T, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("GetOne operation started")

	if r.paused {
		r.logVerbose("GetOne operation skipped - buffer is paused")
		return item, nil
	}

	for !r.isFull && r.r == r.w {
		r.logVerbose("Buffer is empty, handling empty state")
		if err := r.handleBufferEmpty(); err != nil {
			r.logVerbose("GetOne operation failed: %v", err)
			return item, err
		}
	}

	item = r.buf[r.r]
	r.r = (r.r + 1) % r.size
	r.isFull = false

	r.logVerbose("Item read successfully, new read position: %d, isFull: %v", r.r, r.isFull)

	r.notifyReadHandlers(item)

	if r.block && r.blockedWriters.Load() > 0 {
		r.logVerbose("Signaling blocked writers")
		r.readCond.Signal()
	}

	return item, nil
}

// GetN reads n items from the buffer.
// Returns the items read and any error encountered.
func (r *RingBuffer[T]) GetN(n int) (items []T, err error) {
	if n <= 0 {
		return nil, nil
	}

	if n > r.Length(false) {
		n = r.Length(false)
	}

	items = make([]T, 0, n)
	for range n {
		item, err := r.GetOne()
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

// TryWrite attempts to write an item to the buffer without blocking.
// Returns true if successful, false if the buffer is full.
func (r *RingBuffer[T]) TryWrite(item T) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		return false
	}

	if r.isFull {
		if !r.overwrite {
			return false
		}
		r.r = (r.r + 1) % r.size
	}

	r.buf[r.w] = item
	r.w = (r.w + 1) % r.size
	r.isFull = r.w == r.r

	r.notifyWriteHandlers(item)

	if r.block && r.blockedReaders.Load() > 0 {
		r.logVerbose("Signaling blocked readers")
		r.readCond.Signal()
	}
	return true
}

// TryRead attempts to read an item from the buffer without blocking.
// Returns the item and true if successful, false if the buffer is empty.
func (r *RingBuffer[T]) TryRead() (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		var zero T
		return zero, false
	}

	if !r.isFull && r.r == r.w {
		var zero T
		return zero, false
	}

	item := r.buf[r.r]
	r.r = (r.r + 1) % r.size
	r.isFull = false

	r.notifyReadHandlers(item)

	if r.block && r.blockedWriters.Load() > 0 {
		r.writeCond.Signal()
	}
	return item, true
}

// GetAllView returns two slices that together contain all items in the buffer.
// This is useful for zero-copy operations where you need to view all items.
func (r *RingBuffer[T]) GetAllView() (part1, part2 []T, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isFull && r.r == r.w {
		return nil, nil, ErrIsEmpty
	}

	if r.w > r.r {
		return r.buf[r.r:r.w], nil, nil
	}
	// WARNING: doesn't notify

	return r.buf[r.r:], r.buf[:r.w], nil
}

// GetNView returns two slices that together contain the next n items in the buffer.
// This is useful for zero-copy operations where you need to view a specific number of items.
func (r *RingBuffer[T]) GetNView(n int) (part1, part2 []T, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isFull && r.r == r.w {
		return nil, nil, ErrIsEmpty
	}

	length := r.Length(true)
	if n > length {
		n = length
	}

	if r.w > r.r {
		if n <= r.w-r.r {
			return r.buf[r.r : r.r+n], nil, nil
		}
		return r.buf[r.r:r.w], nil, nil
	}

	firstPart := r.size - r.r
	if n <= firstPart {
		return r.buf[r.r : r.r+n], nil, nil
	}

	return r.buf[r.r:], r.buf[:n-firstPart], nil
}

// Flush clears all items from the buffer.
func (r *RingBuffer[T]) Flush() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("Flush operation started")

	r.r = 0
	r.w = 0
	r.isFull = false

	r.logVerbose("Buffer flushed, positions reset to 0")

	if r.onFlushHook != nil {
		r.onFlushHook()
	}

	r.updateBufferState()
	r.broadcastConditions()
}

// Reset resets the buffer to its initial state.
func (r *RingBuffer[T]) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("Reset operation started")

	r.r = 0
	r.w = 0
	r.isFull = false
	r.err = nil
	r.paused = false

	r.logVerbose("Buffer reset to initial state")

	if r.onResetHook != nil {
		r.onResetHook()
	}

	r.updateBufferState()
	r.broadcastConditions()
}

// WaitUntilFull waits until the buffer is full or the timeout is reached.
// Returns true if the buffer became full, false if the timeout was reached.
func (r *RingBuffer[T]) WaitUntilFull(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}

	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		if r.isFull {
			r.mu.Unlock()
			return true
		}

		if time.Now().After(deadline) {
			r.mu.Unlock()
			return false
		}

		r.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
}

// WaitUntilEmpty waits until the buffer is empty or the timeout is reached.
// Returns true if the buffer became empty, false if the timeout was reached.
func (r *RingBuffer[T]) WaitUntilEmpty(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}

	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		if !r.isFull && r.r == r.w {
			r.mu.Unlock()
			return true
		}

		if time.Now().After(deadline) {
			r.mu.Unlock()
			return false
		}

		r.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
}

// Close closes the buffer and releases any resources.
func (r *RingBuffer[T]) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.onCloseHook != nil {
		r.onCloseHook()
	}

	r.r = 0
	r.w = 0
	r.isFull = false
	r.err = nil
	r.paused = true

	r.broadcastConditions()
	return nil
}

// Pause pauses the buffer, preventing any read or write operations.
func (r *RingBuffer[T]) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("Pause operation started")
	r.paused = true

	if r.onPauseHook != nil {
		r.onPauseHook()
	}

	r.logVerbose("Buffer paused")
}

// Resume resumes the buffer, allowing read and write operations to proceed.
func (r *RingBuffer[T]) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logVerbose("Resume operation started")
	r.paused = false

	if r.onResumeHook != nil {
		r.onResumeHook()
	}

	r.logVerbose("Buffer resumed")
}

// Read reads data from the buffer into the provided slice.
// If the buffer is empty and blocking is enabled, it will block until data is available.
// If the buffer is empty and blocking is disabled, it returns ErrIsEmpty.
// Returns the number of items read and any error encountered.
func (r *RingBuffer[T]) Read(data []T) (n int, err error) {
	if len(data) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		return 0, nil
	}

	// If buffer is empty and blocking is enabled, wait for data
	if !r.isFull && r.r == r.w {
		if err := r.handleBufferEmpty(); err != nil {
			return 0, err
		}
	}

	// Calculate how many items we can read
	available := r.Length(true)
	if available == 0 {
		return 0, ErrIsEmpty
	}

	// Limit read to either available items or requested size
	toRead := min(len(data), available)

	// Read items
	for i := range toRead {
		data[i] = r.buf[r.r]
		r.r = (r.r + 1) % r.size
		r.isFull = false
		r.notifyReadHandlers(data[i])
	}

	r.writeCond.Signal()
	return toRead, nil
}
