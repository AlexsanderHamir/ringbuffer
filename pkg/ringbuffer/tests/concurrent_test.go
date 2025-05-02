package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentReadWrite(t *testing.T) {
	const iterations = 5

	for iter := range iterations {
		t.Run(fmt.Sprintf("Iteration_%d", iter), func(t *testing.T) {
			rb := ringbuffer.NewRingBuffer[int](100)
			rb.WithBlocking(true)
			require.NotNil(t, rb)

			const numWriters = 10
			const numReaders = 10
			const itemsPerWriter = 100

			var wg sync.WaitGroup
			results := make(chan int, numWriters*itemsPerWriter)

			// Start writers
			for i := range numWriters {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()
					for j := range itemsPerWriter {
						item := writerID*itemsPerWriter + j
						err := rb.Write(item)
						require.NoError(t, err)
					}
				}(i)
			}

			// Start readers
			for range numReaders {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range itemsPerWriter {
						item, err := rb.GetOne()
						if err != nil {
							break
						}
						results <- item
					}
				}()
			}

			wg.Wait()
			rb.Close()
			close(results)

			received := make(map[int]bool)
			for item := range results {
				received[item] = true
			}

			// Verify all items were received
			for i := range numWriters {
				for j := range itemsPerWriter {
					item := i*itemsPerWriter + j
					assert.True(t, received[item], "Item %d was not received in iteration %d", item, iter)
				}
			}
		})
	}
}

func TestConcurrentBlocking(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	rb.WithBlocking(true)

	const numWriters = 3
	const numReaders = 2
	const timeout = 100 * time.Millisecond

	var wg sync.WaitGroup
	results := make(chan int, 100)

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				item := writerID*10 + j
				err := rb.Write(item)
				if err == nil {
					results <- item
				}
			}
		}(i)
	}

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item, err := rb.GetOne()
				if err != nil {
					break
				}
				results <- item
			}
		}()
	}

	// Wait for a short time
	time.Sleep(timeout)

	// Close the buffer
	rb.Close()

	// Wait for all goroutines to finish
	wg.Wait()
	close(results)

	// Verify we got some results
	assert.Greater(t, len(results), 0, "No results were received")
}

func TestConcurrentPauseResume(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](5)
	require.NotNil(t, rb)

	const numWriters = 2
	const numReaders = 2
	const itemsPerWriter = 10

	var wg sync.WaitGroup
	results := make(chan int, numWriters*itemsPerWriter)

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWriter; j++ {
				item := writerID*itemsPerWriter + j
				err := rb.Write(item)
				if err == nil {
					results <- item
				}
			}
		}(i)
	}

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item, err := rb.GetOne()
				if err != nil {
					break
				}
				results <- item
			}
		}()
	}

	// Pause and resume multiple times
	for i := 0; i < 3; i++ {
		rb.Pause()
		time.Sleep(10 * time.Millisecond)
		rb.Resume()
		time.Sleep(10 * time.Millisecond)
	}

	// Close the buffer
	rb.Close()

	// Wait for all goroutines to finish
	wg.Wait()
	close(results)

	// Verify we got some results
	assert.Greater(t, len(results), 0, "No results were received")
}

func TestConcurrentOverwrite(t *testing.T) {
	rb := ringbuffer.NewRingBuffer[int](3)
	require.NotNil(t, rb)

	rb.WithOverwrite(true)

	const numWriters = 3
	const itemsPerWriter = 10

	var wg sync.WaitGroup
	results := make(chan int, numWriters*itemsPerWriter)

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWriter; j++ {
				item := writerID*itemsPerWriter + j
				err := rb.Write(item)
				if err == nil {
					results <- item
				}
			}
		}(i)
	}

	// Start a reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			item, err := rb.GetOne()
			if err != nil {
				break
			}
			results <- item
		}
	}()

	// Wait for a short time
	time.Sleep(100 * time.Millisecond)

	// Close the buffer
	rb.Close()

	// Wait for all goroutines to finish
	wg.Wait()
	close(results)

	// Verify we got some results
	assert.Greater(t, len(results), 0, "No results were received")
}
