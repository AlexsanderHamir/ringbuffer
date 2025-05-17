# Ring Buffer

This is a **thread-safe ring buffer** that uses **generics** instead of raw `[]byte` as its main type.  
Forked from [smallnest/ringbuffer](https://github.com/smallnest/ringbuffer).

[![GoDoc](https://pkg.go.dev/badge/github.com/AlexsanderHamir/ringbuffer)](https://pkg.go.dev/github.com/AlexsanderHamir/ringbuffer)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexsanderHamir/ringbuffer)](https://goreportcard.com/report/github.com/AlexsanderHamir/ringbuffer)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/AlexsanderHamir/ringbuffer/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexsanderHamir/ringbuffer/actions/workflows/ci.yml)

![Circular Buffer Animation](Circular_Buffer_Animation.gif)

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
    "github.com/AlexsanderHamir/ringbuffer/errors"
    "github.com/AlexsanderHamir/ringbuffer"
    "github.com/AlexsanderHamir/ringbuffer/config"
)

func main() {
    // Create a new ring buffer for integers with size 100
    rb := ringbuffer.New[int](100)

    // Write data to the buffer
    rb.Write(1)
    rb.WriteMany([]int{2, 3, 4, 5})

    // Read data from the buffer
    item, err := rb.GetOne()
    if err != nil {
        fmt.Printf("Error reading: %v\n", err)
        return
    }
    fmt.Println(item) // Output: 1

    items, err := rb.GetN(3)
    if err != nil {
        fmt.Printf("Error reading: %v\n", err)
        return
    }
    fmt.Println(items) // Output: [2 3 4]

    // Example with custom type
    type Message struct {
        ID   int
        Text string
        Time time.Time
    }

    // Create a ring buffer for Message type with custom configuration
    config := &config.RingBufferConfig{
        Block:     true,
        RTimeout:  5 * time.Second,
        WTimeout:  5 * time.Second,
    }

    msgBuffer, err := ringbuffer.NewWithConfig[Message](1000, config)
    if err != nil {
        fmt.Printf("Error creating buffer: %v\n", err)
        return
    }

    // Write messages
    messages := []Message{
        {ID: 1, Text: "Hello", Time: time.Now()},
        {ID: 2, Text: "World", Time: time.Now()},
    }

    _, err = msgBuffer.WriteMany(messages)
    if err != nil {
        fmt.Printf("Error writing messages: %v\n", err)
        return
    }
}
```

## Configuration

The ring buffer can be configured using the `RingBufferConfig` struct:

```go
type RingBufferConfig struct {
    Block            bool          // Enable/disable blocking behavior
    RTimeout         time.Duration // Read operation timeout
    WTimeout         time.Duration // Write operation timeout
    PreReadBlockHook func() bool   // Hook called before blocking on read
    PreWriteBlockHook func() bool  // Hook called before blocking on write
}
```

### Configuration Options

The ring buffer's behavior can be dynamically configured at runtime using the following methods:

- `WithBlocking(block bool)`: Enables or disables blocking behavior
- `WithTimeout(d time.Duration)`: Sets both read and write timeouts
- `WithReadTimeout(d time.Duration)`: Sets the timeout for read operations
- `WithWriteTimeout(d time.Duration)`: Sets the timeout for write operations
- `WithPreReadBlockHook(hook func() bool)`: Sets hook called before blocking on read
- `WithPreWriteBlockHook(hook func() bool)`: Sets hook called before blocking on write

## API Documentation

### Core Operations

- `New[T](size int)` - Creates a new ring buffer with default configuration for type T
- `NewWithConfig[T](size int, config *Config)` - Creates a new ring buffer with custom configuration for type T
- `Write(item T)` - Writes a single item to the buffer
- `WriteMany(items []T)` - Writes multiple items to the buffer
- `GetOne() (item T, err error)` - Reads a single item from the buffer
- `GetN(n int) (items []T, err error)` - Reads n items from the buffer
- `PeekOne() (item T, err error)` - Peeks at data without removing it from the buffer
- `PeekN(n int) (items []T, err error)` - Peeks at n items without removing them from the buffer
- `Close() error` - Closes the buffer and releases resources

### Buffer State Operations

- `IsEmpty() bool` - Checks if the buffer is empty
- `IsFull() bool` - Checks if the buffer is full
- `Length() int` - Returns the number of items in the buffer
- `Capacity() int` - Returns the maximum number of items the buffer can hold
- `Free() int` - Returns the number of elements that can be written without blocking
- `GetBlockedReaders() int` - Returns the number of readers currently blocked
- `GetBlockedWriters() int` - Returns the number of writers currently blocked

### View Operations

View operations provide direct access to the underlying buffer data without copying:

- `GetAllView() (part1, part2 []T, err error)` - Returns two slices containing all items
- `GetNView(n int) (part1, part2 []T, err error)` - Returns two slices containing n items
- `PeekNView(n int) (part1, part2 []T, err error)` - Returns two slices containing n items without removing them

⚠️ **Important**: View operations return references to the actual buffer data. Modifications to these slices will affect the original buffer data. Use with caution and ensure proper synchronization.

### Hook Methods

- `WithPreReadBlockHook(hook func() bool)` - Sets hook called before blocking on read
- `WithPreWriteBlockHook(hook func() bool)` - Sets hook called before blocking on write

## Error Handling

The ring buffer provides comprehensive error handling for various scenarios:

```go
// Example of error handling
rb := ringbuffer.New[int](10)

// Write with error handling
err := rb.Write(1)
if err != nil {
    switch {
    case errors.Is(err, ringbuffer.ErrTooMuchDataToWrite):
        fmt.Println("Data to write exceeds buffer size")
    case errors.Is(err, ringbuffer.ErrTooMuchDataToPeek):
        fmt.Println("Trying to peek more data than available")
    case errors.Is(err, ringbuffer.ErrIsFull):
        fmt.Println("Buffer is full and not blocking")
    case errors.Is(err, ringbuffer.ErrIsEmpty):
        fmt.Println("Buffer is empty and not blocking")
    case errors.Is(err, ringbuffer.ErrIsNotEmpty):
        fmt.Println("Buffer is not empty and not blocking")
    case errors.Is(err, ringbuffer.ErrAcquireLock):
        fmt.Println("Unable to acquire lock on Try operations")
    case errors.Is(err, ringbuffer.ErrInvalidLength):
        fmt.Println("Invalid buffer length")
    case errors.Is(err, ringbuffer.ErrNilBuffer):
        fmt.Println("Operations performed on nil buffer")
    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

The following errors can be returned by the ring buffer operations:

- `ErrTooMuchDataToWrite`: Returned when the data to write is more than the buffer size
- `ErrTooMuchDataToPeek`: Returned when trying to peek more data than available
- `ErrIsFull`: Returned when the buffer is full and not blocking
- `ErrIsEmpty`: Returned when the buffer is empty and not blocking
- `ErrIsNotEmpty`: Returned when the buffer is not empty and not blocking
- `ErrInvalidLength`: Returned when the length of the buffer is invalid
- `ErrNilBuffer`: Returned when operations are performed on a nil buffer

## Performance Considerations

1. **Buffer Size**: Choose an appropriate buffer size based on your use case. Too small buffers may cause frequent blocking, while too large buffers may waste memory.

2. **Blocking vs Non-blocking**: Use non-blocking operations when you need to handle full/empty conditions in your application logic.

3. **View Operations**: Use view operations when memory efficiency is critical, but be aware of the implications of working with direct buffer references.

4. **Hooks**: Use hooks if there's something you can do before blocking, but keep hook functions lightweight to avoid impacting performance.

5. **Time Out**: At high concurrency many timer structs will be created, it's responsible for almost 90% of memory consumption.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
