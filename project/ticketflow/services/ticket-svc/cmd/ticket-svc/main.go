// Command ticket-svc consumes order.created and issues tickets.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abhiraj860/ticketflow/pkg/config"
	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/blob"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/generator"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/issuer"
)

const serviceName = "ticket-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	l := config.New("TICKET")
	var (
		httpAddr    = l.String("HTTP_ADDR", ":9122")
		brokers     = l.String("KAFKA_BROKERS", "localhost:9092")
		groupID     = l.String("KAFKA_GROUP", "ticket-issuer")
		concurrency = l.Int("CONCURRENCY", 8)
		batchSize   = l.Int("BATCH_SIZE", 20)
		bucket      = l.String("S3_BUCKET", "ticketflow-tickets")
		s3Endpoint  = l.String("S3_ENDPOINT", "http://localhost:4566")
		s3Region    = l.String("S3_REGION", "us-east-1")
		s3Key       = l.String("S3_ACCESS_KEY", "test")
		s3Secret    = l.String("S3_SECRET_KEY", "test")
		fsRoot      = l.String("FS_ROOT", "")
		signingKey  = l.String("SIGNING_KEY", "dev-signing-key-not-for-production-use!!")
		logLevel    = l.String("LOG_LEVEL", "info")
		logFormat   = l.String("LOG_FORMAT", "json")
	)
	if err := l.Err(); err != nil {
		return err
	}

	logger := logging.New(logging.Options{Service: serviceName, Level: logLevel, Format: logging.Format(logFormat)})
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// A filesystem store when FS_ROOT is set, so the pipeline runs without
	// LocalStack; S3 otherwise.
	var store blob.Store
	if fsRoot != "" {
		fs, err := blob.NewFSStore(fsRoot)
		if err != nil {
			return err
		}
		store = fs
		logger.Info("using filesystem blob store", slog.String("root", fsRoot))
	} else {
		s3store, err := blob.NewS3Store(ctx, blob.S3Options{
			Bucket: bucket, Region: s3Region, Endpoint: s3Endpoint,
			AccessKey: s3Key, SecretKey: s3Secret,
		})
		if err != nil {
			return fmt.Errorf("connecting to s3: %w", err)
		}
		if err := s3store.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("ensuring bucket: %w", err)
		}
		store = s3store
		logger.Info("using s3 blob store",
			slog.String("bucket", bucket), slog.String("endpoint", s3Endpoint))
	}

	gen, err := generator.New(generator.Options{SigningKey: []byte(signingKey)})
	if err != nil {
		return err
	}

	producer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{Brokers: strings.Split(brokers, ",")})
	if err != nil {
		return fmt.Errorf("kafka producer: %w", err)
	}
	defer func() { _ = producer.Close() }()

	iss := issuer.New(issuer.Options{
		Generator: gen, Store: store, Publisher: producer, Logger: logger,
	})

	consumer, err := tfkafka.NewConsumer(tfkafka.ConsumerOptions{
		Brokers: strings.Split(brokers, ","),
		Topic:   tfkafka.TopicOrderCreated,
		GroupID: groupID,
		// PDF rendering is CPU-bound with no I/O to overlap, so the pool size
		// is what determines throughput during a drop.
		Concurrency: concurrency,
		BatchSize:   batchSize,
		MaxAttempts: 3,
		DLQTopic:    tfkafka.TopicDLQ,
		DLQProducer: producer,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("kafka consumer: %w", err)
	}
	defer func() { _ = consumer.Close() }()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		s, c := iss.Stats(), consumer.Stats()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "ticket_issued_total %d\n", s.Issued)
		fmt.Fprintf(w, "ticket_skipped_total %d\n", s.Skipped)
		fmt.Fprintf(w, "ticket_failed_total %d\n", s.Failed)
		fmt.Fprintf(w, "ticket_consumer_processed_total %d\n", c.Processed)
		fmt.Fprintf(w, "ticket_consumer_retried_total %d\n", c.Retried)
		fmt.Fprintf(w, "ticket_consumer_dead_lettered_total %d\n", c.DeadLettered)
	})
	httpSrv := &http.Server{Addr: httpAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		logger.Info("http listening", slog.String("addr", httpAddr))
		_ = httpSrv.ListenAndServe()
	}()

	logger.Info("consuming", slog.String("topic", tfkafka.TopicOrderCreated))
	if err := consumer.Run(ctx, iss.Handle); err != nil {
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	logger.Info("shutdown complete")
	return nil
}
