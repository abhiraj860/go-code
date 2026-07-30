// Command inventory-svc serves the inventory gRPC API.
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
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/abhiraj860/ticketflow/pkg/config"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	"github.com/abhiraj860/ticketflow/pkg/postgres"
	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	inventory "github.com/abhiraj860/ticketflow/services/inventory-svc"
	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/grpcserver"
	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/repo"
	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/service"
)

const serviceName = "inventory-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.Migrate(cfg.dbDSN, inventory.Migrations, inventory.MigrationsDir); err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	version, dirty, err := postgres.Version(cfg.dbDSN, inventory.Migrations, inventory.MigrationsDir)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	if dirty {
		return fmt.Errorf("schema is dirty at version %d; manual intervention required", version)
	}
	logger.Info("schema ready", slog.Uint64("version", uint64(version)))

	// Inventory is the write hot path during a drop, so it gets a larger pool
	// than catalog. N replicas * MaxConns must still stay under the server's
	// max_connections.
	pool, err := postgres.Connect(ctx, postgres.Options{
		DSN:      cfg.dbDSN,
		MaxConns: int32(cfg.dbMaxConns),
	})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	svc := service.New(service.Options{
		Repo:       repo.NewSeatRepo(pool),
		Logger:     logger,
		DefaultTTL: cfg.holdTTL,
		MaxTTL:     cfg.maxHoldTTL,
	})

	// The reaper is an availability-freshness optimisation, not a correctness
	// requirement -- the hold predicate already treats lapsed holds as claimable.
	go svc.RunReaper(ctx, cfg.reaperInterval, cfg.reaperBatchSize)

	grpcServer := grpc.NewServer()
	inventoryv1.RegisterInventoryServiceServer(grpcServer, grpcserver.New(svc))

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.grpcAddr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	httpSrv := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           mux,
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

	// Stop taking new work before draining, so a blue/green flip loses nothing.
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
		logger.Warn("grpc drain timed out, forcing stop")
		grpcServer.Stop()
	}

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown", slog.Any("error", err))
	}

	logger.Info("shutdown complete")
	return nil
}

type appConfig struct {
	grpcAddr        string
	httpAddr        string
	dbDSN           string
	dbMaxConns      int
	holdTTL         time.Duration
	maxHoldTTL      time.Duration
	reaperInterval  time.Duration
	reaperBatchSize int
	shutdownTimeout time.Duration
	logLevel        string
	logFormat       string
}

func loadConfig() (appConfig, error) {
	l := config.New("INVENTORY")

	cfg := appConfig{
		grpcAddr:        l.String("GRPC_ADDR", ":9111"),
		httpAddr:        l.String("HTTP_ADDR", ":9112"),
		dbDSN:           l.Required("DB_DSN"),
		dbMaxConns:      l.Int("DB_MAX_CONNS", 20),
		holdTTL:         l.Duration("HOLD_TTL", 2*time.Minute),
		maxHoldTTL:      l.Duration("MAX_HOLD_TTL", 10*time.Minute),
		reaperInterval:  l.Duration("REAPER_INTERVAL", 10*time.Second),
		reaperBatchSize: l.Int("REAPER_BATCH_SIZE", 1000),
		shutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		logLevel:        l.String("LOG_LEVEL", "info"),
		logFormat:       l.String("LOG_FORMAT", "json"),
	}

	return cfg, l.Err()
}
