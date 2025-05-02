package ringbuffer

// WithPreReadBlockHook sets a hook function that will be called before blocking on a read
// or hitting a deadline. This allows for custom handling of blocking situations,
// such as trying alternative sources for data.
func (r *RingBuffer[T]) WithPreReadBlockHook(hook func() bool) *RingBuffer[T] {
	r.mu.Lock()
	r.preReadBlockHook = hook
	r.mu.Unlock()
	return r
}

// WithPreWriteBlockHook sets a hook function that will be called before blocking on a write
// or hitting a deadline. This allows for custom handling of blocking situations,
// such as trying alternative destinations for data.
func (r *RingBuffer[T]) WithPreWriteBlockHook(hook func() bool) *RingBuffer[T] {
	r.mu.Lock()
	r.preWriteBlockHook = hook
	r.mu.Unlock()
	return r
}

// WithOnFullHook sets a hook function that will be called when the buffer becomes full.
func (r *RingBuffer[T]) WithOnFullHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onFullHook = hook
	r.mu.Unlock()
	return r
}

// WithOnEmptyHook sets a hook function that will be called when the buffer becomes empty.
func (r *RingBuffer[T]) WithOnEmptyHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onEmptyHook = hook
	r.mu.Unlock()
	return r
}

// WithOnWriteHook sets a hook function that will be called after each write attempt.
// The hook receives the item that was written and whether the write was successful.
func (r *RingBuffer[T]) WithOnWriteHook(hook func(item T, success bool)) *RingBuffer[T] {
	r.mu.Lock()
	r.onWriteHook = hook
	r.mu.Unlock()
	return r
}

// WithOnReadHook sets a hook function that will be called after each read attempt.
// The hook receives the item that was read and whether the read was successful.
func (r *RingBuffer[T]) WithOnReadHook(hook func(item T, success bool)) *RingBuffer[T] {
	r.mu.Lock()
	r.onReadHook = hook
	r.mu.Unlock()
	return r
}

// WithOnResetHook sets a hook function that will be called after the buffer is reset.
func (r *RingBuffer[T]) WithOnResetHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onResetHook = hook
	r.mu.Unlock()
	return r
}

// WithOnFlushHook sets a hook function that will be called after the buffer is flushed.
func (r *RingBuffer[T]) WithOnFlushHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onFlushHook = hook
	r.mu.Unlock()
	return r
}

// WithOnCloseHook sets a hook function that will be called before the buffer is closed.
func (r *RingBuffer[T]) WithOnCloseHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onCloseHook = hook
	r.mu.Unlock()
	return r
}

// WithOnErrorHook sets a hook function that will be called when an error occurs.
func (r *RingBuffer[T]) WithOnErrorHook(hook func(err error)) *RingBuffer[T] {
	r.mu.Lock()
	r.onErrorHook = hook
	r.mu.Unlock()
	return r
}

// WithOnStateChangeHook sets a hook function that will be called when the buffer state changes.
func (r *RingBuffer[T]) WithOnStateChangeHook(hook func(isFull bool, length int)) *RingBuffer[T] {
	r.mu.Lock()
	r.onStateChangeHook = hook
	r.mu.Unlock()
	return r
}

// WithOnPauseHook sets a hook function that will be called when the buffer is paused.
func (r *RingBuffer[T]) WithOnPauseHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onPauseHook = hook
	r.mu.Unlock()
	return r
}

// WithOnResumeHook sets a hook function that will be called when the buffer is resumed.
func (r *RingBuffer[T]) WithOnResumeHook(hook func()) *RingBuffer[T] {
	r.mu.Lock()
	r.onResumeHook = hook
	r.mu.Unlock()
	return r
}
