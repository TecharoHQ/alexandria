//go:build limitedsupportability

package alexandria

import (
	"io"
	"log/slog"
	"os"
)

func Writer(kind string, logID string, next io.Writer) *WriterWrapper {
	result := &WriterWrapper{
		next: next,
	}

	lg := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	}))

	lg.Info("Logging to Alexandria has been disabled by build tag. Your ability to recieve support is limited.", "docs", "https://anubis.techaro.lol/docs/admin/alexandria")

	return result
}

type WriterWrapper struct {
	next io.Writer
}

func (ww *WriterWrapper) SetBaseURL(baseURL string) {}

func (ww *WriterWrapper) Write(data []byte) (n int, err error) {
	return ww.next.Write(data)
}
