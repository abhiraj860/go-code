// Command gateway-bff serves the public REST API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/abhiraj860/ticketflow/pkg/config"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	"github.com/abhiraj860/ticketflow/services/gateway-bff/internal/httpapi"
	"github.com/abhiraj860/ticketflow/services/gateway-bff/internal/session"
)

const serviceName = "gateway-bff"

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

	// grpc.NewClient is lazy: it does not block on a connection, so the BFF
	// starts even if a downstream service is still coming up. Requests fail
	// with Unavailable until it is ready, which is the right behaviour during
	// a rolling deploy.
	catalogConn, err := grpc.NewClient(cfg.catalogAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling catalog at %s: %w", cfg.catalogAddr, err)
	}
	defer func() { _ = catalogConn.Close() }()

	inventoryConn, err := grpc.NewClient(cfg.inventoryAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling inventory at %s: %w", cfg.inventoryAddr, err)
	}
	defer func() { _ = inventoryConn.Close() }()

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.redisAddr,
		// DB 2 is sessions. Cache is DB 0, seat locks DB 1 -- separated so an
		// evicting maxmemory policy on the cache can never sign users out.
		DB:           cfg.redisDB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		// Sessions degrade rather than fail: browsing works without one, and
		// only the hold endpoints require it.
		logger.Warn("redis unavailable, sessions will not persist",
			slog.String("addr", cfg.redisAddr), slog.Any("error", err))
	}

	api := httpapi.New(httpapi.Options{
		Catalog:         catalogv1.NewCatalogServiceClient(catalogConn),
		Inventory:       inventoryv1.NewInventoryServiceClient(inventoryConn),
		Sessions:        session.NewStore(session.Options{Client: redisClient, TTL: cfg.sessionTTL}),
		Logger:          logger,
		UpstreamTimeout: cfg.upstreamTimeout,
	})

	if cfg.logLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery(), requestLogger(logger))
	api.Register(router)

	srv := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// Generous relative to upstreamTimeout so a slow upstream surfaces as a
		// 504 from our own handler rather than a truncated connection.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  90 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http listening", slog.String("addr", cfg.httpAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	// Shutdown stops accepting new connections and waits for in-flight requests,
	// which is what makes a blue/green flip lose nothing.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown", slog.Any("error", err))
	}

	logger.Info("shutdown complete")
	return nil
}

// requestLogger emits one structured line per request. Written by hand rather
// than using gin.Logger() so the output is JSON and CloudWatch Logs Insights
// can query the fields.
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		level := slog.LevelInfo
		if c.Writer.Status() >= http.StatusInternalServerError {
			level = slog.LevelError
		}

		logger.LogAttrs(c.Request.Context(), level, "request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
		)
	}
}

type appConfig struct {
	httpAddr        string
	catalogAddr     string
	inventoryAddr   string
	redisAddr       string
	redisDB         int
	sessionTTL      time.Duration
	upstreamTimeout time.Duration
	shutdownTimeout time.Duration
	logLevel        string
	logFormat       string
}

func loadConfig() (appConfig, error) {
	l := config.New("BFF")

	cfg := appConfig{
		httpAddr:        l.String("HTTP_ADDR", ":8080"),
		catalogAddr:     l.String("CATALOG_ADDR", "localhost:9101"),
		inventoryAddr:   l.String("INVENTORY_ADDR", "localhost:9111"),
		redisAddr:       l.String("REDIS_ADDR", "localhost:6379"),
		redisDB:         l.Int("REDIS_DB", 2),
		sessionTTL:      l.Duration("SESSION_TTL", 24*time.Hour),
		upstreamTimeout: l.Duration("UPSTREAM_TIMEOUT", 2*time.Second),
		shutdownTimeout: l.Duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		logLevel:        l.String("LOG_LEVEL", "info"),
		logFormat:       l.String("LOG_FORMAT", "json"),
	}

	return cfg, l.Err()
}
