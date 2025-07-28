package alexandria

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewRingBuffer(t *testing.T) {
	rb := newRingBuffer()

	if rb == nil {
		t.Fatal("newRingBuffer() returned nil")
	}

	if len(rb.buffer) != ringBufferSize {
		t.Errorf("expected buffer size %d, got %d", ringBufferSize, len(rb.buffer))
	}

	if rb.head != 0 || rb.tail != 0 || rb.count != 0 {
		t.Errorf("expected initial state head=0, tail=0, count=0, got head=%d, tail=%d, count=%d",
			rb.head, rb.tail, rb.count)
	}
}

func TestRingBuffer_Add(t *testing.T) {
	tests := []struct {
		name          string
		itemsToAdd    [][]byte
		expectedCount int
		expectedHead  int
		expectedTail  int
	}{
		{
			name:          "add single item",
			itemsToAdd:    [][]byte{[]byte("test1")},
			expectedCount: 1,
			expectedHead:  1,
			expectedTail:  0,
		},
		{
			name:          "add multiple items",
			itemsToAdd:    [][]byte{[]byte("test1"), []byte("test2"), []byte("test3")},
			expectedCount: 3,
			expectedHead:  3,
			expectedTail:  0,
		},
		{
			name:          "fill buffer exactly",
			itemsToAdd:    generateTestData(ringBufferSize),
			expectedCount: ringBufferSize,
			expectedHead:  0,
			expectedTail:  0,
		},
		{
			name:          "overflow buffer by one",
			itemsToAdd:    generateTestData(ringBufferSize + 1),
			expectedCount: ringBufferSize,
			expectedHead:  1,
			expectedTail:  1,
		},
		{
			name:          "overflow buffer significantly",
			itemsToAdd:    generateTestData(ringBufferSize + 100),
			expectedCount: ringBufferSize,
			expectedHead:  100,
			expectedTail:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := newRingBuffer()

			for _, item := range tt.itemsToAdd {
				rb.add(item)
			}

			if rb.count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, rb.count)
			}

			if rb.head != tt.expectedHead {
				t.Errorf("expected head %d, got %d", tt.expectedHead, rb.head)
			}

			if rb.tail != tt.expectedTail {
				t.Errorf("expected tail %d, got %d", tt.expectedTail, rb.tail)
			}
		})
	}
}

func TestRingBuffer_Drain(t *testing.T) {
	tests := []struct {
		name           string
		setupItems     [][]byte
		expectedResult [][]byte
		expectedCount  int
		expectedHead   int
		expectedTail   int
	}{
		{
			name:           "drain empty buffer",
			setupItems:     nil,
			expectedResult: nil,
			expectedCount:  0,
			expectedHead:   0,
			expectedTail:   0,
		},
		{
			name:           "drain single item",
			setupItems:     [][]byte{[]byte("test1")},
			expectedResult: [][]byte{[]byte("test1")},
			expectedCount:  0,
			expectedHead:   0,
			expectedTail:   0,
		},
		{
			name:           "drain multiple items",
			setupItems:     [][]byte{[]byte("test1"), []byte("test2"), []byte("test3")},
			expectedResult: [][]byte{[]byte("test1"), []byte("test2"), []byte("test3")},
			expectedCount:  0,
			expectedHead:   0,
			expectedTail:   0,
		},
		{
			name:           "drain full buffer",
			setupItems:     generateTestData(ringBufferSize),
			expectedResult: generateTestData(ringBufferSize),
			expectedCount:  0,
			expectedHead:   0,
			expectedTail:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := newRingBuffer()

			// Setup
			for _, item := range tt.setupItems {
				rb.add(item)
			}

			// Test drain
			result := rb.drain()

			// Check result
			if !equalByteSlices(result, tt.expectedResult) {
				t.Errorf("drain result mismatch.\nExpected: %v\nGot: %v", tt.expectedResult, result)
			}

			// Check buffer state after drain
			if rb.count != tt.expectedCount {
				t.Errorf("expected count after drain %d, got %d", tt.expectedCount, rb.count)
			}

			if rb.head != tt.expectedHead {
				t.Errorf("expected head after drain %d, got %d", tt.expectedHead, rb.head)
			}

			if rb.tail != tt.expectedTail {
				t.Errorf("expected tail after drain %d, got %d", tt.expectedTail, rb.tail)
			}
		})
	}
}

func TestRingBuffer_OverflowBehavior(t *testing.T) {
	rb := newRingBuffer()

	// Fill buffer
	for i := 0; i < ringBufferSize; i++ {
		rb.add([]byte{byte(i)})
	}

	// Add one more to trigger overflow
	rb.add([]byte("overflow"))

	// Drain and verify oldest item was overwritten
	result := rb.drain()

	if len(result) != ringBufferSize {
		t.Errorf("expected %d items after overflow, got %d", ringBufferSize, len(result))
	}

	// First item should be byte(1), not byte(0) (which was overwritten)
	if !bytes.Equal(result[0], []byte{1}) {
		t.Errorf("expected first item to be [1], got %v", result[0])
	}

	// Last item should be "overflow"
	if !bytes.Equal(result[len(result)-1], []byte("overflow")) {
		t.Errorf("expected last item to be 'overflow', got %v", result[len(result)-1])
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := newRingBuffer()
	const numGoroutines = 10
	const itemsPerGoroutine = 100

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				data := []byte{byte(id), byte(j)}
				rb.add(data)
			}
		}(i)
	}

	wg.Wait()

	// Verify buffer state is consistent
	rb.mu.RLock()
	count := rb.count
	head := rb.head
	tail := rb.tail
	rb.mu.RUnlock()

	if count < 0 || count > ringBufferSize {
		t.Errorf("invalid count after concurrent adds: %d", count)
	}

	if head < 0 || head >= ringBufferSize {
		t.Errorf("invalid head after concurrent adds: %d", head)
	}

	if tail < 0 || tail >= ringBufferSize {
		t.Errorf("invalid tail after concurrent adds: %d", tail)
	}

	// Drain should not panic and should return valid data
	result := rb.drain()
	if result != nil && len(result) != count {
		t.Errorf("drain returned %d items but count was %d", len(result), count)
	}
}

func TestRingBuffer_AddAfterDrain(t *testing.T) {
	rb := newRingBuffer()

	// Add some items
	rb.add([]byte("test1"))
	rb.add([]byte("test2"))

	// Drain
	result1 := rb.drain()
	expectedFirst := [][]byte{[]byte("test1"), []byte("test2")}
	if !equalByteSlices(result1, expectedFirst) {
		t.Errorf("first drain mismatch. Expected: %v, Got: %v", expectedFirst, result1)
	}

	// Add more items after drain
	rb.add([]byte("test3"))
	rb.add([]byte("test4"))

	// Drain again
	result2 := rb.drain()
	expectedSecond := [][]byte{[]byte("test3"), []byte("test4")}
	if !equalByteSlices(result2, expectedSecond) {
		t.Errorf("second drain mismatch. Expected: %v, Got: %v", expectedSecond, result2)
	}
}

// Helper functions
func generateTestData(count int) [][]byte {
	result := make([][]byte, count)
	for i := 0; i < count; i++ {
		result[i] = []byte{byte(i % 256)}
	}
	return result
}

func equalByteSlices(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}
