// Command search-svc indexes catalog events and serves faceted search.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/abhiraj860/ticketflow/pkg/config"
	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	"github.com/abhiraj860/ticketflow/services/search-svc/internal/index"
	"github.com/abhiraj860/ticketflow/services/search-svc/internal/indexer"
	"github.com/abhiraj860/ticketflow/services/search-svc/internal/search"
)

const serviceName = "search-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	l := config.New("SEARCH")
	var (
		httpAddr    = l.String("HTTP_ADDR", ":9132")
		esURL       = l.String("ELASTICSEARCH_URL", "http://localhost:9200")
		brokers     = l.String("KAFKA_BROKERS", "localhost:9092")
		groupID     = l.String("KAFKA_GROUP", "search-indexer")
		catalogAddr = l.String("CATALOG_ADDR", "localhost:9101")
		concurrency = l.Int("CONCURRENCY", 4)
		batchSize   = l.Int("BATCH_SIZE", 50)
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

	esClient, err := index.NewClient(esURL)
	if err != nil {
		return err
	}
	if err := esClient.Ping(ctx); err != nil {
		return fmt.Errorf("elasticsearch unreachable: %w", err)
	}
	if err := esClient.EnsureIndex(ctx); err != nil {
		return err
	}
	logger.Info("index ready", slog.String("index", index.Name), slog.String("url", esURL))

	catalogConn, err := grpc.NewClient(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling catalog: %w", err)
	}
	defer func() { _ = catalogConn.Close() }()

	idx := indexer.New(catalogv1.NewCatalogServiceClient(catalogConn), esClient, logger)
	searcher := search.New(esClient)

	producer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{Brokers: strings.Split(brokers, ",")})
	if err != nil {
		return fmt.Errorf("kafka producer: %w", err)
	}
	defer func() { _ = producer.Close() }()

	consumer, err := tfkafka.NewConsumer(tfkafka.ConsumerOptions{
		Brokers: strings.Split(brokers, ","),
		Topic:   tfkafka.TopicCatalogEventUpdated,
		GroupID: groupID,
		// Modest concurrency and a large batch: indexing is I/O-bound on
		// ElasticSearch, not CPU-bound like PDF rendering, so batching helps
		// more than parallelism.
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

	go func() {
		if err := consumer.Run(ctx, idx.Handle); err != nil {
			logger.Error("indexer stopped", slog.Any("error", err))
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		s, c := idx.Stats(), consumer.Stats()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "search_indexed_total %d\n", s.Indexed)
		fmt.Fprintf(w, "search_deleted_total %d\n", s.Deleted)
		fmt.Fprintf(w, "search_index_failed_total %d\n", s.Failed)
		fmt.Fprintf(w, "search_consumer_processed_total %d\n", c.Processed)
		fmt.Fprintf(w, "search_consumer_dead_lettered_total %d\n", c.DeadLettered)
	})
	mux.HandleFunc("GET /v1/search", func(w http.ResponseWriter, r *http.Request) {
		q := search.Query{
			Text:       r.URL.Query().Get("q"),
			City:       r.URL.Query().Get("city"),
			OnSaleOnly: r.URL.Query().Get("on_sale") == "true",
		}
		if k := r.URL.Query().Get("kind"); k != "" {
			if n, err := strconv.Atoi(k); err == nil {
				q.Kind = int16(n)
			}
		}
		if t := r.URL.Query().Get("tags"); t != "" {
			q.Tags = strings.Split(t, ",")
		}
		if s := r.URL.Query().Get("size"); s != "" {
			q.Size, _ = strconv.Atoi(s)
		}
		if f := r.URL.Query().Get("from"); f != "" {
			q.From, _ = strconv.Atoi(f)
		}

		reqCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		result, err := searcher.Search(reqCtx, q)
		if err != nil {
			logger.ErrorContext(reqCtx, "search failed", slog.Any("error", err))
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "search unavailable"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// Search results change only on reindex, so a short shared cache is
		// safe and absorbs the repeated queries a facet sidebar generates.
		w.Header().Set("Cache-Control", "public, max-age=30")
		_ = json.NewEncoder(w).Encode(result)
	})

	httpSrv := &http.Server{Addr: httpAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("http listening", slog.String("addr", httpAddr))
		_ = httpSrv.ListenAndServe()
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	logger.Info("shutdown complete")
	return nil
}
