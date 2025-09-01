package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var knownKinds = []string{
	"techaro.anubis",
	"techaro.thoth",
}

type Server struct {
	s3c *s3.Client
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
	buf := bytes.NewBuffer(data)

	id := uuid.Must(uuid.NewV7()).String()

	if _, err := s.s3c.PutObject(ctx, &s3.PutObjectInput{
		Body:        buf,
		Bucket:      bucket,
		Key:         aws.String(fmt.Sprintf("logs/%s/%s/%s.jsonl", kind, logID, id)),
		ContentType: aws.String("application/jsonl"),
	}); err != nil {
		return err
	}

	return nil
}
