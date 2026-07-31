// Command order-svc serves the order API and runs the outbox relay.
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
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/abhiraj860/ticketflow/pkg/config"
	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	"github.com/abhiraj860/ticketflow/pkg/postgres"
	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	orderv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/order/v1"
	order "github.com/abhiraj860/ticketflow/services/order-svc"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/grpcserver"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/relay"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/repo"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/service"
)

const serviceName = "order-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	l := config.New("ORDER")
	var (
		grpcAddr      = l.String("GRPC_ADDR", ":9141")
		httpAddr      = l.String("HTTP_ADDR", ":9142")
		dbDSN         = l.Required("DB_DSN")
		dbMaxConns    = l.Int("DB_MAX_CONNS", 15)
		inventoryAddr = l.String("INVENTORY_ADDR", "localhost:9111")
		brokers       = l.String("KAFKA_BROKERS", "localhost:9092")
		relayInterval = l.Duration("RELAY_INTERVAL", 200*time.Millisecond)
		relayBatch    = l.Int("RELAY_BATCH_SIZE", 100)
		shutdownGrace = l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second)
		logLevel      = l.String("LOG_LEVEL", "info")
		logFormat     = l.String("LOG_FORMAT", "json")
	)
	if err := l.Err(); err != nil {
		return err
	}

	logger := logging.New(logging.Options{Service: serviceName, Level: logLevel, Format: logging.Format(logFormat)})
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.Migrate(dbDSN, order.Migrations, order.MigrationsDir); err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	version, dirty, err := postgres.Version(dbDSN, order.Migrations, order.MigrationsDir)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	if dirty {
		return fmt.Errorf("schema is dirty at version %d; manual intervention required", version)
	}
	logger.Info("schema ready", slog.Uint64("version", uint64(version)))

	pool, err := postgres.Connect(ctx, postgres.Options{DSN: dbDSN, MaxConns: int32(dbMaxConns)})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	inventoryConn, err := grpc.NewClient(inventoryAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling inventory: %w", err)
	}
	defer func() { _ = inventoryConn.Close() }()

	orderRepo := repo.NewOrderRepo(pool)
	svc := service.New(service.Options{
		Repo:      orderRepo,
		Inventory: inventoryv1.NewInventoryServiceClient(inventoryConn),
		Logger:    logger,
	})

	// The relay runs in-process rather than as a separate deployment. It shares
	// the database connection and lifecycle, and FOR UPDATE SKIP LOCKED means
	// every replica can run one safely -- so scaling order-svc scales the relay
	// with it, and there is no second thing to deploy or forget to restart.
	producer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{Brokers: strings.Split(brokers, ",")})
	if err != nil {
		return fmt.Errorf("kafka producer: %w", err)
	}
	defer func() { _ = producer.Close() }()

	outboxRelay := relay.New(relay.Options{
		Store: orderRepo, Publisher: producer, Logger: logger,
		Interval: relayInterval, BatchSize: relayBatch,
	})
	go outboxRelay.Run(ctx)

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcserver.New(svc))

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", grpcAddr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		s := outboxRelay.Stats(r.Context())
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "order_outbox_published_total %d\n", s.Published)
		fmt.Fprintf(w, "order_outbox_failed_batches_total %d\n", s.FailedBatches)
		fmt.Fprintf(w, "order_outbox_sweeps_total %d\n", s.Sweeps)
		// The metric that matters most: a climbing backlog means orders are
		// being accepted that nothing is fulfilling.
		fmt.Fprintf(w, "order_outbox_pending %d\n", s.Pending)
	})
	httpSrv := &http.Server{Addr: httpAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("grpc listening", slog.String("addr", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()
	go func() {
		logger.Info("http listening", slog.String("addr", httpAddr))
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

	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
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
		logger.Warn("grpc drain timed out, forcing stop")
		grpcServer.Stop()
	}

	_ = httpSrv.Shutdown(shutdownCtx)
	logger.Info("shutdown complete")
	return nil
}
