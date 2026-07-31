// Command catalog-svc serves the catalog gRPC API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/abhiraj860/ticketflow/pkg/cache"
	"github.com/abhiraj860/ticketflow/pkg/config"
	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	"github.com/abhiraj860/ticketflow/pkg/postgres"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	catalog "github.com/abhiraj860/ticketflow/services/catalog-svc"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/grpcserver"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/invalidator"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/repo"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/service"
)

const serviceName = "catalog-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

// run returns an error instead of calling os.Exit so every deferred cleanup
// actually runs -- os.Exit skips defers.
func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger := logging.New(logging.Options{
		Service: serviceName,
		Level:   cfg.logLevel,
		Format:  logging.Format(cfg.logFormat),
	})
	slog.SetDefault(logger)

	// Signal-aware root context: SIGTERM (what Kubernetes and ECS send) cancels
	// it, which unwinds every in-flight operation below.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("applying migrations")
	if err := postgres.Migrate(cfg.dbDSN, catalog.Migrations, catalog.MigrationsDir); err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	version, dirty, err := postgres.Version(cfg.dbDSN, catalog.Migrations, catalog.MigrationsDir)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	if dirty {
		// A half-applied schema needs a human. Serving traffic against it would
		// produce errors that look like application bugs.
		return fmt.Errorf("schema is dirty at version %d; manual intervention required", version)
	}
	logger.Info("schema ready", slog.Uint64("version", uint64(version)))

	pool, err := postgres.Connect(ctx, postgres.Options{
		DSN:      cfg.dbDSN,
		MaxConns: int32(cfg.dbMaxConns),
	})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	// Redis is the L2 cache tier. It is an optimisation, not a dependency: if
	// it is unreachable the service still serves correctly from Postgres, just
	// slower. Refusing to boot over a cache would be the wrong trade.
	var l2 cache.Store
	redisStore, err := cache.NewRedisStore(ctx, cache.RedisOptions{
		Addr: cfg.redisAddr,
		DB:   cfg.redisDB,
	})
	if err != nil {
		logger.Warn("redis unavailable, running with L1 cache only",
			slog.String("addr", cfg.redisAddr), slog.Any("error", err))
	} else {
		defer func() { _ = redisStore.Close() }()
		l2 = redisStore
	}

	// Mongo holds editorial content. Like Redis it is treated as optional:
	// content is enrichment, and an event page without a setlist is still a
	// usable event page. Refusing to boot over it would be the wrong trade.
	var content service.ContentRepo
	mongoClient, err := repo.ConnectMongo(ctx, cfg.mongoURI, cfg.mongoTimeout)
	if err != nil {
		logger.Warn("mongo unavailable, serving without event content",
			slog.Any("error", err))
	} else {
		defer func() { _ = mongoClient.Disconnect(context.Background()) }()
		content = repo.NewContentRepo(mongoClient)
	}

	svc := service.New(service.Options{
		Repo:       repo.NewEventRepo(pool),
		Content:    content,
		L2:         l2,
		EventTTL:   cfg.eventTTL,
		SeatMapTTL: cfg.seatMapTTL,
		L2TTL:      cfg.l2TTL,
	})

	// Cache invalidation over Kafka. Each replica needs its OWN consumer group:
	// a shared group would deliver each message to exactly one replica, when
	// every replica must hear it to drop its local copy. The instance id makes
	// the group unique per process.
	var inv *invalidator.Invalidator
	if cfg.kafkaBrokers != "" {
		dlqProducer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{
			Brokers: strings.Split(cfg.kafkaBrokers, ","),
		})
		if err != nil {
			logger.Warn("kafka unavailable, L1 caches will only expire on TTL",
				slog.Any("error", err))
		} else {
			defer func() { _ = dlqProducer.Close() }()

			consumer, err := tfkafka.NewConsumer(tfkafka.ConsumerOptions{
				Brokers: strings.Split(cfg.kafkaBrokers, ","),
				Topic:   tfkafka.TopicCatalogEventUpdated,
				GroupID: "catalog-invalidator-" + instanceID(),
				// Serial: invalidation is a map delete, so parallelism would
				// add contention without adding throughput.
				Concurrency: 1,
				BatchSize:   50,
				MaxAttempts: 2,
				DLQTopic:    tfkafka.TopicDLQ,
				DLQProducer: dlqProducer,
				Logger:      logger,
			})
			if err != nil {
				logger.Warn("could not start invalidation consumer", slog.Any("error", err))
			} else {
				defer func() { _ = consumer.Close() }()
				inv = invalidator.New(consumer, svc, logger)
				go func() {
					if err := inv.Run(ctx); err != nil {
						logger.Error("invalidation consumer stopped", slog.Any("error", err))
					}
				}()
			}
		}
	}

	grpcServer := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcServer, grpcserver.New(svc))

	// Standard gRPC health service, so Kubernetes probes and ECS target groups
	// need no custom check.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Reflection lets grpcurl explore the API without a .proto file on hand --
	// invaluable while debugging, and harmless behind a private network.
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.grpcAddr, err)
	}

	// Separate plain-HTTP listener for liveness and cache metrics; gRPC and
	// HTTP cannot share a port without extra multiplexing machinery.
	httpSrv := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           adminMux(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 2)

	go func() {
		logger.Info("grpc listening", slog.String("addr", cfg.grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	go func() {
		logger.Info("http listening", slog.String("addr", cfg.httpAddr))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// Report NOT_SERVING before draining so load balancers stop sending new
	// work while in-flight requests finish. This is what makes a blue/green
	// flip lose zero requests.
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("grpc drained")
	case <-shutdownCtx.Done():
		// A stuck stream must not hold the process open past its grace period,
		// or the orchestrator SIGKILLs us mid-request instead.
		logger.Warn("grpc drain timed out, forcing stop")
		grpcServer.Stop()
	}

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown", slog.Any("error", err))
	}

	logger.Info("shutdown complete")
	return nil
}

// instanceID returns a per-process identifier, used to give this replica its
// own Kafka consumer group. Hostname in Kubernetes is the pod name; the PID
// keeps two local processes distinct during development.
func instanceID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

// adminMux serves liveness and cache statistics.
func adminMux(svc *service.Catalog) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Cache hit ratios, which Phase 7 uses to tune TTLs. Prometheus-style text
	// so this endpoint can be scraped directly later.
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		stats := svc.CacheStats()
		// Sorted so the output is stable between scrapes, which makes diffing
		// two captures during an incident actually readable.
		names := make([]string, 0, len(stats))
		for name := range stats {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			writeCacheMetrics(w, name, stats[name])
		}
	})

	return mux
}

func writeCacheMetrics(w http.ResponseWriter, name string, s cache.LoaderStats) {
	fmt.Fprintf(w, "catalog_cache_l1_hits_total{cache=%q} %d\n", name, s.L1Hits)
	fmt.Fprintf(w, "catalog_cache_l2_hits_total{cache=%q} %d\n", name, s.L2Hits)
	fmt.Fprintf(w, "catalog_cache_fetches_total{cache=%q} %d\n", name, s.Fetches)
	fmt.Fprintf(w, "catalog_cache_negative_hits_total{cache=%q} %d\n", name, s.NegativeHits)
	fmt.Fprintf(w, "catalog_cache_coalesced_total{cache=%q} %d\n", name, s.Coalesced)
	fmt.Fprintf(w, "catalog_cache_errors_total{cache=%q} %d\n", name, s.Errors)
}

type appConfig struct {
	grpcAddr        string
	httpAddr        string
	dbDSN           string
	dbMaxConns      int
	redisAddr       string
	redisDB         int
	mongoURI        string
	mongoTimeout    time.Duration
	kafkaBrokers    string
	eventTTL        time.Duration
	seatMapTTL      time.Duration
	l2TTL           time.Duration
	shutdownTimeout time.Duration
	logLevel        string
	logFormat       string
}

func loadConfig() (appConfig, error) {
	l := config.New("CATALOG")

	cfg := appConfig{
		grpcAddr:   l.String("GRPC_ADDR", ":9101"),
		httpAddr:   l.String("HTTP_ADDR", ":9102"),
		dbDSN:      l.Required("DB_DSN"),
		dbMaxConns: l.Int("DB_MAX_CONNS", 10),
		redisAddr:  l.String("REDIS_ADDR", "localhost:6379"),
		// DB 0 is the cache database; seat-hold locks live in DB 1 so an
		// evicting maxmemory policy can never discard a lock.
		redisDB:         l.Int("REDIS_DB", 0),
		mongoURI:        l.String("MONGO_URI", "mongodb://ticketflow:ticketflow@localhost:27017/?authSource=admin"),
		mongoTimeout:    l.Duration("MONGO_TIMEOUT", 5*time.Second),
		kafkaBrokers:    l.String("KAFKA_BROKERS", "localhost:9092"),
		eventTTL:        l.Duration("EVENT_TTL", 30*time.Second),
		seatMapTTL:      l.Duration("SEAT_MAP_TTL", 10*time.Minute),
		l2TTL:           l.Duration("L2_TTL", time.Hour),
		shutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		logLevel:        l.String("LOG_LEVEL", "info"),
		logFormat:       l.String("LOG_FORMAT", "json"),
	}

	return cfg, l.Err()
}
