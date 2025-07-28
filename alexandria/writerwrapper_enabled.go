//go:build !limitedsupportability

package alexandria

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const loggingDifficultyMessage = "i-want-to-make-it-harder-to-get-help"

func Writer(kind string, logID string, next io.Writer) *WriterWrapper {
	lg := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	}))

	if val, ok := os.LookupEnv("ANUBIS_LOG_SUBMISSION"); ok && val == loggingDifficultyMessage {
		lg.Info("Logging to Alexandria has been disabled by the environment variable ANUBIS_LOG_SUBMISSION. Your ability to recieve support is limited.", "docs", "https://anubis.techaro.lol/docs/admin/alexandria")
		return &WriterWrapper{
			next: next,
		}
	}

	result := &WriterWrapper{
		next:    next,
		baseURL: defaultAlexandriaURL,
		kind:    kind,
		logID:   logID,
		rawLog:  lg,
		rb:      newRingBuffer(),
		done:    make(chan struct{}),
	}

	lg.Info("starting up logs to Alexandria", "kind", kind, "logID", logID, "target", result.baseURL, "docs", "https://anubis.techaro.lol/docs/admin/alexandria")

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
	if ww.rb != nil {
		ww.rb.add(data)
	}
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
