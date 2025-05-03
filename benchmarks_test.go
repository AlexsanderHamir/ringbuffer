package ringbuffer

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkWrite tests write performance with different buffer sizes
func BenchmarkWrite(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rb := New[int](size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rb.Write(i)
			}
		})
	}
}

// BenchmarkRead tests read performance with different buffer sizes
func BenchmarkRead(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rb := New[int](size)
			// Pre-fill the buffer
			for i := range size {
				rb.Write(i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rb.GetOne()
			}
		})
	}
}

// BenchmarkConcurrentRW tests concurrent read/write performance
func BenchmarkConcurrentRW(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rb := New[int](size)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if rb.IsFull() {
						rb.GetOne()
					} else {
						rb.Write(1)
					}
				}
			})
		})
	}
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			rb := New[int](size)

			// Test allocation patterns during write
			b.Run("Write", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.Write(i)
				}
			})

			// Test allocation patterns during read
			b.Run("GetOne", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.GetOne()
				}
			})
		})
	}
}

// BenchmarkBlockingOperations tests performance with blocking enabled
func BenchmarkBlockingOperations(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rb := New[int](size)
			rb.WithBlocking(true)
			rb.WithTimeout(100 * time.Millisecond)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if rb.IsFull() {
						rb.GetOne()
					} else {
						rb.Write(1)
					}
				}
			})
		})
	}
}

// BenchmarkBufferOperations tests various buffer operations
func BenchmarkBufferOperations(b *testing.B) {
	sizes := []int{64, 1024, 8192, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rb := New[int](size)

			// Test Length() operation
			b.Run("Length", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.Length(true)
				}
			})

			// Test Free() operation
			b.Run("Free", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.Free()
				}
			})

			// Test IsFull() operation
			b.Run("IsFull", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.IsFull()
				}
			})

			// Test IsEmpty() operation
			b.Run("IsEmpty", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rb.IsEmpty()
				}
			})
		})
	}
}
