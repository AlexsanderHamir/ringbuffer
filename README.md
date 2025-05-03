# Ring Buffer

A high-performance, thread-safe ring buffer implementation in Go that leverages the power of generics. Built with Go's type system in mind, this implementation allows you to create type-safe ring buffers for any data type - from simple primitives to complex structs. The generic implementation ensures compile-time type safety while maintaining the flexibility to work with any data type you need.

This library provides a robust and efficient circular buffer data structure that can be used in various scenarios where you need to handle data streams, implement queues, or manage fixed-size buffers.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Error Handling](#error-handling)
- [Performance Considerations](#performance-considerations)
- [Contributing](#contributing)
- [License](#license)

## Features

- Generic type support
- Built-in hooks
- Configurable blocking behavior
- Timeout support for operations
- Overwrite mode for full buffers
- Memory-efficient view operations

## Installation

```bash
go get github.com/AlexsanderHamir/ringbuffer
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/AlexsanderHamir/ringbuffer"
)

func main() {
    // Create a new ring buffer for integers with size 100
    rb := ringbuffer.New[int](100)

    // Write data to the buffer
    rb.Write([]int{1, 2, 3, 4, 5})

    // Read data from the buffer
    data := make([]int, 5)
    rb.Read(data)
    fmt.Println(data) // Output: [1 2 3 4 5]

    // Example with custom type
    type Message struct {
        ID   int
        Text string
        Time time.Time
    }

    // Create a ring buffer for Message type with custom configuration
    config := &ringbuffer.RingBufferConfig{
        Block:     true,
        RTimeout:  5 * time.Second,
        WTimeout:  5 * time.Second,
        Overwrite: true,
    }
    msgBuffer := ringbuffer.NewWithConfig[Message](1000, config)

    // Write messages
    messages := []Message{
        {ID: 1, Text: "Hello", Time: time.Now()},
        {ID: 2, Text: "World", Time: time.Now()},
    }
    msgBuffer.Write(messages)

    // Read messages with timeout
    readMessages := make([]Message, 2)
    err := msgBuffer.Read(readMessages)
    if err != nil {
        fmt.Printf("Error reading messages: %v\n", err)
        return
    }
}
```

## Configuration

The ring buffer can be configured using the `Config` struct:

```go
type RingBufferConfig struct {
    Block     bool          // Enable/disable blocking behavior
    RTimeout  time.Duration // Read operation timeout
    WTimeout  time.Duration // Write operation timeout
    Overwrite bool          // Allows overwriting existing data when buffer is full
}
```

### Configuration Options

The ring buffer's behavior can be dynamically configured at runtime using the following methods:

- `WithBlocking(block bool)`: Enables or disables blocking behavior
- `WithTimeout(d time.Duration)`: Sets both read and write timeouts
- `WithReadTimeout(d time.Duration)`: Sets the timeout for read operations
- `WithWriteTimeout(d time.Duration)`: Sets the timeout for write operations
- `WithOverwrite(overwrite bool)`: Enables or disables overwriting data when the buffer is full

### Example Configuration

```go
// Create a new ring buffer with custom configuration
config := &ringbuffer.RingBufferConfig{
    Block:     true,
    RTimeout:  5 * time.Second,
    WTimeout:  5 * time.Second,
    Overwrite: true,
}
rb := ringbuffer.NewWithConfig[int](100, config)

// Or configure an existing buffer
rb.WithBlocking(true)
rb.WithTimeout(5 * time.Second)
rb.WithOverwrite(true)
```

## API Documentation

### Core Operations

- `New[T](size int)` - Creates a new ring buffer with default configuration for type T
- `NewWithConfig[T](size int, config *Config)` - Creates a new ring buffer with custom configuration for type T
- `Write(item T)` - Writes a single item of type T to the buffer
- `WriteMany(items []T)` - Writes multiple items to the buffer
- `Read(data []T)` - Reads data of type T from the buffer into provided slice
- `GetOne() (item T, err error)` - Reads a single item from the buffer
- `GetN(n int) (items []T, err error)` - Reads n items from the buffer
- `PeekOne() (item T, err error)` - Peeks at data without removing it from the buffer
- `PeekN(n int) (items []T, err error)` - Peeks at n items without removing them from the buffer
- `TryWrite(item T) bool` - Attempts to write an item without blocking
- `TryRead() (T, bool)` - Attempts to read an item without blocking
- `Reset()` - Resets the buffer to its initial state
- `Flush()` - Clears all items from the buffer
- `Close() error` - Closes the buffer and releases resources

### Buffer State Operations

- `IsEmpty() bool` - Checks if the buffer is empty
- `IsFull() bool` - Checks if the buffer is full
- `IsPaused() bool` - Checks if the buffer is paused
- `Length() int` - Returns the number of items in the buffer
- `Capacity() int` - Returns the maximum number of items the buffer can hold
- `Free() int` - Returns the number of elements available for writing
- `GetBlockedReaders() int` - Returns the number of readers currently blocked
- `GetBlockedWriters() int` - Returns the number of writers currently blocked

### Control Operations

- `Pause()` - Pauses the buffer, preventing read/write operations
- `Resume()` - Resumes the buffer, allowing read/write operations
- `WaitUntilFull(timeout time.Duration) bool` - Waits until buffer is full or timeout
- `WaitUntilEmpty(timeout time.Duration) bool` - Waits until buffer is empty or timeout

### View Operations

View operations provide direct access to the underlying buffer data without copying:

- `GetAllView() (part1, part2 []T, err error)` - Returns two slices containing all items
- `GetNView(n int) (part1, part2 []T, err error)` - Returns two slices containing n items

⚠️ **Important**: View operations return references to the actual buffer data. Modifications to these slices will affect the original buffer data. Use with caution and ensure proper synchronization.

### Hook Methods

- `WithPreReadBlockHook(hook func() bool)` - Sets hook called before blocking on read
- `WithPreWriteBlockHook(hook func() bool)` - Sets hook called before blocking on write
- `WithOnFullHook(hook func())` - Sets hook called when buffer becomes full
- `WithOnEmptyHook(hook func())` - Sets hook called when buffer becomes empty
- `WithOnWriteHook(hook func(item T, success bool))` - Sets hook called after write
- `WithOnReadHook(hook func(item T, success bool))` - Sets hook called after read
- `WithOnResetHook(hook func())` - Sets hook called after buffer reset
- `WithOnFlushHook(hook func())` - Sets hook called after buffer flush
- `WithOnCloseHook(hook func())` - Sets hook called before buffer close
- `WithOnErrorHook(hook func(err error))` - Sets hook called when error occurs
- `WithOnStateChangeHook(hook func(isFull bool, length int))` - Sets hook for state changes
- `WithOnPauseHook(hook func())` - Sets hook called when buffer is paused
- `WithOnResumeHook(hook func())` - Sets hook called when buffer is resumed

## Error Handling

The ring buffer provides comprehensive error handling for various scenarios:

```go
// Example of error handling
rb := ringbuffer.New[int](10)

// Write with error handling
err := rb.Write([]int{1, 2, 3})
if err != nil {
    switch {
    case errors.Is(err, ringbuffer.ErrBufferClosed):
        fmt.Println("Buffer is closed")
    case errors.Is(err, ringbuffer.ErrBufferFull):
        fmt.Println("Buffer is full")
    case errors.Is(err, ringbuffer.ErrBufferPaused):
        fmt.Println("Buffer is paused")
    case errors.Is(err, ringbuffer.ErrTimeout):
        fmt.Println("Operation timed out")
    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
}

// Read with error handling
data := make([]int, 3)
err = rb.Read(data)
if err != nil {
    // Handle read errors
}
```

## Performance Considerations

1. **Buffer Size**: Choose an appropriate buffer size based on your use case. Too small buffers may cause frequent blocking, while too large buffers may waste memory.

2. **Blocking vs Non-blocking**: Use non-blocking operations (`TryRead`/`TryWrite`) when you need to handle full/empty conditions in your application logic.

3. **View Operations**: Use view operations when memory efficiency is critical, but be aware of the implications of working with direct buffer references.

4. **Hooks**: Use hooks for monitoring and control, but keep hook functions lightweight to avoid impacting performance.

5. **Concurrent Access**: The buffer is thread-safe, but consider using appropriate synchronization mechanisms when sharing the buffer between goroutines.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
