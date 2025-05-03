package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentReadWrite(t *testing.T) {
	const iterations = 5

	for iter := range iterations {
		t.Run(fmt.Sprintf("Iteration_%d", iter), func(t *testing.T) {
			rb := ringbuffer.New[int](100)
			rb.WithBlocking(true)
			require.NotNil(t, rb)

			const numWriters = 100
			const numReaders = 100
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

func TestConcurrentNoBlocking(t *testing.T) {
	const iterations = 5

	for iter := range iterations {
		t.Run(fmt.Sprintf("Iteration_%d", iter), func(t *testing.T) {
			rb := ringbuffer.New[int](10)
			require.NotNil(t, rb)

			const numWriters = 200
			const numReaders = 200
			const itemsPerWriter = 20

			var wg sync.WaitGroup
			results := make(chan int, numWriters*itemsPerWriter)
			errorsChan := make(chan error, numWriters*itemsPerWriter)

			// Start writers
			for i := range numWriters {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()
					for j := range itemsPerWriter {
						item := writerID*itemsPerWriter + j
						err := rb.Write(item)
						if err != nil && err != errors.ErrIsFull {
							errorsChan <- err
							continue
						}
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
						if err != nil && err != errors.ErrIsEmpty {
							errorsChan <- err
							continue
						}
						results <- item
					}
				}()
			}

			wg.Wait()
			rb.Close()
			close(results)
			close(errorsChan)

			// Verify no unexpected errors occurred
			assert.Equal(t, 0, len(errorsChan), "Expected no unexpected errors")

			// Verify received items
			received := make(map[int]bool)
			for item := range results {
				received[item] = true
			}

			// Verify we received at least some items
			assert.Greater(t, len(received), 0, "Expected to receive some items")
		})
	}
}

func TestConcurrentReadTimeout(t *testing.T) {
	const iterations = 5

	for iter := range iterations {
		t.Run(fmt.Sprintf("Iteration_%d", iter), func(t *testing.T) {
			rb := ringbuffer.New[int](10)
			require.NotNil(t, rb)

			var wg sync.WaitGroup
			const numReaders = 10
			errorsChan := make(chan error, numReaders)
			rb.WithReadTimeout(100 * time.Millisecond)

			for range numReaders {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := rb.GetOne()
					if err != context.DeadlineExceeded {
						errorsChan <- err
					}
				}()
			}

			wg.Wait()
			rb.Close()
			close(errorsChan)

			// Verify no unexpected errors occurred
			assert.Equal(t, 0, len(errorsChan), "Expected no unexpected errors")
		})
	}
}

func TestConcurrentWriteTimeout(t *testing.T) {
	const iterations = 5

	for iter := range iterations {
		t.Run(fmt.Sprintf("Iteration_%d", iter), func(t *testing.T) {
			rb := ringbuffer.New[int](1)
			rb.WithWriteTimeout(100 * time.Millisecond)
			require.NotNil(t, rb)

			err := rb.Write(1)
			require.NoError(t, err)

			const numWriters = 5
			var wg sync.WaitGroup
			errorsChan := make(chan error, numWriters)

			for i := range numWriters {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()
					item := writerID
					err := rb.Write(item)
					if err != context.DeadlineExceeded {
						errorsChan <- err
					}
				}(i)
			}

			wg.Wait()
			rb.Close()
			close(errorsChan)

			// Verify no unexpected errors occurred
			assert.Equal(t, 0, len(errorsChan), "Expected no unexpected errors")
		})
	}
}
