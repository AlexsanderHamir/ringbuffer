# Ring Buffer

[![GoDoc](https://pkg.go.dev/badge/github.com/AlexsanderHamir/ringbuffer)](https://pkg.go.dev/github.com/AlexsanderHamir/ringbuffer)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexsanderHamir/ringbuffer)](https://goreportcard.com/report/github.com/AlexsanderHamir/ringbuffer)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/AlexsanderHamir/ringbuffer/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexsanderHamir/ringbuffer/actions/workflows/ci.yml)

![Circular Buffer Animation](Circular_Buffer_Animation.gif)

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
    config := &ringbuffer.RingBufferConfig{
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

- `New[T](size int)`
