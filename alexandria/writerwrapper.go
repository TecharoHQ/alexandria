package alexandria

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultAlexandriaURL = "https://alexandria.probably-not-malware.lol"
	ringBufferSize       = 1024
	flushInterval        = time.Second
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

func Writer(kind string, logID string, next io.Writer) *WriterWrapper {
	result := &WriterWrapper{
		next:    next,
		baseURL: defaultAlexandriaURL,
		kind:    kind,
		logID:   logID,
		rawLog: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
		})),
		rb:   newRingBuffer(),
		done: make(chan struct{}),
	}

	result.rawLog.Info("starting up logs to alexandria", "kind", kind, "logID", logID, "target", result.baseURL)

	// Start the background flush goroutine
	go result.flushLoop()

	return result
}

type WriterWrapper struct {
	rb      *ringBuffer
	next    io.Writer
	baseURL string
	kind    string
	logID   string
	rawLog  *slog.Logger
	done    chan struct{}
}

func (ww *WriterWrapper) SetBaseURL(baseURL string) {
	ww.baseURL = baseURL
}

func (ww *WriterWrapper) Write(data []byte) (n int, err error) {
	ww.rb.add(data)
	return ww.next.Write(data)
}

func (ww *WriterWrapper) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ww.flush()
		case <-ww.done:
			// Final flush before exit
			ww.flush()
			return
		}
	}
}

func (ww *WriterWrapper) Close() error {
	close(ww.done)
	return nil
}

func (ww *WriterWrapper) flush() {
	lines := ww.rb.drain()
	if len(lines) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), flushInterval)
	defer cancel()

	ww.submit(ctx, lines)
}

func (ww *WriterWrapper) submit(ctx context.Context, lines [][]byte) {
	buf := bytes.NewBuffer(nil)

	for _, line := range lines {
		buf.Write(line)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/upload/%s/%s", ww.baseURL, ww.kind, ww.logID), buf)
	if err != nil {
		ww.rawLog.Error("can't create request to alexandria", "err", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ww.rawLog.Error("can't perform request to alexandria", "err", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		ww.rawLog.Error("wrong alexandria response code", "status", resp.StatusCode, "want", http.StatusOK)
		return
	}
}
