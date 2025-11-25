package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"within.website/x/bundler"
)

var knownKinds = []string{
	"techaro.anubis",
	"techaro.anubis.request-samples",
	"techaro.thoth",
}

// LogEntry represents a single log entry in the batch
type LogEntry struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	LogID string `json:"logID"`
	Data  string `json:"data"`
}

type Server struct {
	s3c      *s3.Client
	bundlers map[string]*bundler.Bundler[LogEntry]
}

// NewServer creates a new Server with configured bundlers for each kind
func NewServer(s3c *s3.Client, bucket string) *Server {
	s := &Server{
		s3c:      s3c,
		bundlers: make(map[string]*bundler.Bundler[LogEntry]),
	}

	// Create a bundler for each known kind
	for _, kind := range knownKinds {
		b := bundler.New[LogEntry](func(ctx context.Context, items []LogEntry) {
			if err := s.uploadBatch(ctx, bucket, items); err != nil {
				slog.Error("failed to upload batch", "kind", kind, "err", err)
			}
		})

		// Configure bundler thresholds
		b.DelayThreshold = 2 * time.Minute // 2 minutes
		b.ContextDeadline = time.Minute
		b.BundleCountThreshold = 0       // no limit on number of items
		b.BundleByteThreshold = 32 << 20 // 32MiB
		b.BundleByteLimit = 32 << 20     // 32MiB max bundle size
		b.BufferedByteLimit = 64 << 20   // 64MiB buffer limit
		b.HandlerLimit = 1               // one handler at a time

		s.bundlers[kind] = b
	}

	return s
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {}

func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	logID := r.PathValue("logID")
	slog.Info("got request for", "kind", kind, "logID", logID)

	if !slices.Contains(knownKinds, kind) {
		slog.Error("unknown kind", "kind", kind)
		return
	}

	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("can't read from client", "err", err)
		return
	}

	if err := s.uploadFor(r.Context(), kind, logID, data); err != nil {
		slog.Error("can't publish logs", "err", err)
		return
	}
}

func (s *Server) uploadFor(ctx context.Context, kind, logID string, data []byte) error {
	// Get the bundler for this specific kind
	bundler, exists := s.bundlers[kind]
	if !exists {
		return fmt.Errorf("no bundler found for kind: %s", kind)
	}

	// Create log entry with UUIDv7, kind, logID, and base64 encoded data
	id := uuid.Must(uuid.NewV7()).String()
	encodedData := base64.StdEncoding.EncodeToString(data)

	entry := LogEntry{
		ID:    id,
		Kind:  kind,
		LogID: logID,
		Data:  encodedData,
	}

	// Add to the kind-specific bundler - the size is the length of the JSON representation
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	return bundler.Add(entry, len(jsonData))
}

// uploadBatch handles a batch of log entries, writing them as a JSONL file to S3
func (s *Server) uploadBatch(ctx context.Context, bucket string, items []LogEntry) error {
	if len(items) == 0 {
		return nil
	}

	// Create a buffer for the JSONL content
	var buf bytes.Buffer

	// Write each log entry as a separate line
	for _, item := range items {
		jsonLine, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}
		buf.Write(jsonLine)
		buf.WriteByte('\n')
	}

	// Generate a unique filename for this batch using the first item's info
	batchID := uuid.Must(uuid.NewV7()).String()
	kind := items[0].Kind // Use the kind from the first item

	// Upload the batch to S3
	key := fmt.Sprintf("inp/%s/batch-%s.jsonl", kind, batchID)
	_, err := s.s3c.PutObject(ctx, &s3.PutObjectInput{
		Body:        &buf,
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/jsonl"),
	})

	if err != nil {
		return fmt.Errorf("failed to upload batch to S3: %w", err)
	}

	slog.Info("uploaded batch of logs", "batchID", batchID, "kind", kind, "items", len(items), "size", buf.Len(), "key", key)
	return nil
}
