package alexandria

import (
	"bytes"
	"sync"
	"time"
)

const (
	defaultAlexandriaURL = "https://alexandria.probably-not-malware.lol"
	ringBufferSize       = 1024
	flushInterval        = 5 * time.Second
)

// ringBuffer is a simple ring buffer for storing log entries
type ringBuffer struct {
	mu     sync.RWMutex
	buffer [][]byte
	head   int
	tail   int
	count  int
}

func newRingBuffer() *ringBuffer {
	return &ringBuffer{
		buffer: make([][]byte, ringBufferSize),
	}
}

func (rb *ringBuffer) add(data []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == ringBufferSize {
		// Buffer is full, overwrite oldest entry
		rb.tail = (rb.tail + 1) % ringBufferSize
	} else {
		rb.count++
	}

	rb.buffer[rb.head] = bytes.Clone(data)
	rb.head = (rb.head + 1) % ringBufferSize
}

func (rb *ringBuffer) drain() [][]byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	result := make([][]byte, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.tail + i) % ringBufferSize
		result[i] = rb.buffer[idx]
	}

	rb.count = 0
	rb.head = 0
	rb.tail = 0

	return result
}
