package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/facebookgo/flagenv"
	_ "github.com/joho/godotenv/autoload"
)

var (
	bind   = flag.String("bind", ":8989", "host:port to bind http to")
	bucket = flag.String("bucket", "techaro-anubis-logs", "bucket to store logs into")
)

const maxLogSize = 2 << 16 // 65536 bytes should be enough for anyone

func main() {
	flagenv.Parse()
	flag.Parse()

	ctx := context.Background()

	mux := http.NewServeMux()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create an S3 client
	s3Client := s3.NewFromConfig(cfg)

	s := &Server{
		s3c: s3Client,
	}

	mux.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))

	mux.Handle("PUT /upload/{kind}/{logID}", http.MaxBytesHandler(http.HandlerFunc(s.Upload), maxLogSize))

	slog.Info("listening over HTTP", "bind", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}
