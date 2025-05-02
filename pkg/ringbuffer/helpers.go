package ringbuffer

import (
	"context"
	"log"
	"time"
)

// notifyStateChange handles all state change notifications
func (r *RingBuffer[T]) notifyStateChange(locked bool) {
	if r.onStateChangeHook != nil {
		r.logVerbose("notifying state change: isFull=%v, length=%d", r.isFull, r.Length(locked))
		r.onStateChangeHook(r.isFull, r.Length(locked))
	}
}

// notifyWriteHandlers handles all write-related notifications
func (r *RingBuffer[T]) notifyWriteHandlers(item T) {
	r.logVerbose("notifying write handlers for item: %v", item)
	if r.onWriteHook != nil {
		r.logVerbose("calling onWriteHook for item: %v", item)
		r.onWriteHook(item, true)
	}

	if r.isFull && r.onFullHook != nil {
		r.logVerbose("buffer is full, calling full hook")
		r.onFullHook()
	}

	r.notifyStateChange(true)

	r.logVerbose("notified write handlers for item: %v", item)
}

// notifyReadHandlers handles all read-related notifications
func (r *RingBuffer[T]) notifyReadHandlers(item T) {
	r.logVerbose("notifying read handlers for item: %v", item)
	if r.onReadHook != nil {
		r.onReadHook(item, true)
	}

	if r.r == r.w && r.onEmptyHook != nil {
		r.logVerbose("buffer is empty, calling empty hook")
		r.onEmptyHook()
	}

	r.notifyStateChange(true)
}

// waitForReadCondition handles waiting for read condition with timeout
func (r *RingBuffer[T]) waitForReadCondition() error {
	r.logVerbose("waiting for read condition with timeout: %v", r.rTimeout)
	if r.rTimeout > 0 {
		done := make(chan struct{})
		go func() {
			r.readCond.Wait()
			close(done)
		}()

		select {
		case <-done:
			r.logVerbose("read condition satisfied")
		case <-time.After(r.rTimeout):
			r.logVerbose("read condition timeout exceeded")
			r.blockedReaders.Add(^uint32(0))
			return context.DeadlineExceeded
		}
	} else {
		r.logVerbose("read condition blocked (no timeout)")
		r.readCond.Wait()
		r.logVerbose("read condition satisfied (no timeout)")
	}

	return nil
}

// waitForWriteCondition handles waiting for write condition with timeout
func (r *RingBuffer[T]) waitForWriteCondition() error {
	r.logVerbose("waiting for write condition with timeout: %v", r.wTimeout)
	if r.wTimeout > 0 {
		done := make(chan struct{})
		go func() {
			r.writeCond.Wait()
			close(done)
		}()

		select {
		case <-done:
			r.logVerbose("write condition satisfied")
		case <-time.After(r.wTimeout):
			r.logVerbose("write condition timeout exceeded")
			r.blockedWriters.Add(^uint32(0))
			return context.DeadlineExceeded
		}
	} else {
		r.logVerbose("write condition blocked (no timeout)")
		r.writeCond.Wait()
		r.logVerbose("write condition satisfied (no timeout)")
	}

	return nil
}

// checkPreReadBlock checks if read should be blocked based on pre-read hook
func (r *RingBuffer[T]) checkPreReadBlock() bool {
	if r.preReadBlockHook != nil && r.preReadBlockHook() {
		r.logVerbose("pre-read block hook returned true")
		r.mu.Lock()
		r.blockedReaders.Add(^uint32(0))
		return true
	}
	return false
}

// checkPreWriteBlock checks if write should be blocked based on pre-write hook
func (r *RingBuffer[T]) checkPreWriteBlock() bool {
	if r.preWriteBlockHook != nil && r.preWriteBlockHook() {
		r.logVerbose("pre-write block hook returned true")
		r.mu.Lock()
		r.blockedWriters.Add(^uint32(0))
		return true
	}
	return false
}

// handleBufferFull handles the case when buffer is full
func (r *RingBuffer[T]) handleBufferFull() error {
	r.logVerbose("handling buffer full condition: block=%v, overwrite=%v", r.block, r.overwrite)
	if !r.block {
		return ErrIsFull
	}

	if r.overwrite {
		r.logVerbose("overwriting oldest item due to full buffer")
		r.r = (r.r + 1) % r.size
		if r.r == r.w {
			r.isFull = true
		}
		return nil
	}

	
	r.mu.Unlock()
	if r.checkPreWriteBlock() {
		return nil
	}
	r.mu.Lock()
	
	r.blockedWriters.Add(1)
	if err := r.waitForReadCondition(); err != nil {
		return err
	}
	r.blockedWriters.Add(^uint32(0))
	
	return nil
}

// handleBufferEmpty handles the case when buffer is empty
func (r *RingBuffer[T]) handleBufferEmpty() error {
	r.logVerbose("handling buffer empty condition: block=%v", r.block)
	if !r.block {
		return ErrIsEmpty
	}

	r.blockedReaders.Add(1)
	r.mu.Unlock()
	if r.checkPreReadBlock() {
		r.logVerbose("pre-read block hook returned true")
		return ErrIsEmpty
	}
	r.mu.Lock()

	if err := r.waitForWriteCondition(); err != nil {
		r.logVerbose("error waiting for write condition: %v", err)
		return err
	}

	r.blockedReaders.Add(^uint32(0))
	r.logVerbose("handleBufferEmpty done")
	return nil
}

// updateBufferState updates the buffer state after operations
func (r *RingBuffer[T]) updateBufferState() {
	r.isFull = r.w == r.r
	r.logVerbose("updating buffer state: isFull=%v, read=%d, write=%d", r.isFull, r.r, r.w)
	r.notifyStateChange(true)
}

// broadcastConditions broadcasts to all waiting goroutines
func (r *RingBuffer[T]) broadcastConditions() {
	r.logVerbose("broadcasting conditions to all waiting goroutines")
	r.writeCond.Broadcast()
	r.readCond.Broadcast()
	r.logVerbose("broadcastConditions done")
}

// logVerbose logs a message if verbose mode is enabled
func (r *RingBuffer[T]) logVerbose(format string, args ...interface{}) {
	if r.verbose {
		log.Printf("[RingBuffer] "+format, args...)
	}
}
