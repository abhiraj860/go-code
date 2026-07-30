package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const adminDSN = "postgres://ticketflow:ticketflow@localhost:5432/ticketflow?sslmode=disable"

// requirePostgres skips when no stack is running, so `go test ./...` stays
// green on a machine without Docker.
func requirePostgres(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = adminDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pool, err := Connect(ctx, Options{DSN: dsn, ConnectTimeout: time.Second})
	if err != nil {
		t.Skipf("postgres not reachable (run `make up`): %v", err)
	}
	pool.Close()
	return dsn
}

// newScratchDB creates a throwaway database and returns its DSN, so migration
// tests never collide with a developer's working schema.
func newScratchDB(t *testing.T) string {
	t.Helper()
	admin := requirePostgres(t)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Fatalf("connecting as admin: %v", err)
	}
	defer pool.Close()

	name := fmt.Sprintf("tf_test_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating scratch database: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		p, err := pgxpool.New(cleanupCtx, admin)
		if err != nil {
			return
		}
		defer p.Close()
		// Terminate stragglers so DROP cannot block on a lingering connection.
		_, _ = p.Exec(cleanupCtx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", name)
		_, _ = p.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+name)
	})

	return fmt.Sprintf(
		"postgres://ticketflow:ticketflow@localhost:5432/%s?sslmode=disable", name)
}

func TestConnectRequiresDSN(t *testing.T) {
	if _, err := Connect(context.Background(), Options{}); err == nil {
		t.Error("Connect accepted an empty DSN")
	}
}

func TestConnectFailsFastOnBadDSN(t *testing.T) {
	_, err := Connect(context.Background(), Options{
		DSN:            "postgres://nobody:nobody@127.0.0.1:1/nothing?sslmode=disable",
		ConnectTimeout: 500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Connect succeeded against a dead address")
	}
}

func TestConnectAppliesPoolDefaults(t *testing.T) {
	dsn := requirePostgres(t)

	pool, err := Connect(context.Background(), Options{DSN: dsn})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()

	cfg := pool.Config()
	if cfg.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want default 10", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Errorf("MinConns = %d, want default 2", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour {
		t.Errorf("MaxConnLifetime = %v, want default 1h", cfg.MaxConnLifetime)
	}
}

func TestMigrateAppliesCatalogSchema(t *testing.T) {
	dsn := newScratchDB(t)

	// The real catalog migrations, read from disk the same way embed.FS would
	// present them at runtime.
	fsys := os.DirFS("../../services/catalog-svc")

	if err := Migrate(dsn, fsys, "migrations"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	version, dirty, err := Version(dsn, fsys, "migrations")
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
	if dirty {
		t.Error("schema reported dirty after a clean migration")
	}

	// Confirm the tables actually exist, not merely that migrate recorded a version.
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to scratch db: %v", err)
	}
	defer pool.Close()

	for _, table := range []string{"venue", "seat_map", "seat", "event", "pricing_tier"} {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			                 WHERE table_schema='public' AND table_name=$1)`,
			table).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %q: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q was not created", table)
		}
	}
}

// A second replica booting during a rolling deploy must find nothing to do
// rather than erroring on an already-applied schema.
func TestMigrateIsIdempotent(t *testing.T) {
	dsn := newScratchDB(t)
	fsys := os.DirFS("../../services/inventory-svc")

	if err := Migrate(dsn, fsys, "migrations"); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(dsn, fsys, "migrations"); err != nil {
		t.Fatalf("second Migrate should be a no-op, got: %v", err)
	}

	version, dirty, err := Version(dsn, fsys, "migrations")
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if version != 1 || dirty {
		t.Errorf("version/dirty = %d/%v, want 1/false", version, dirty)
	}
}

func TestVersionOnUnmigratedDatabase(t *testing.T) {
	dsn := newScratchDB(t)

	version, dirty, err := Version(dsn, os.DirFS("../../services/catalog-svc"), "migrations")
	if err != nil {
		t.Fatalf("Version on a fresh database errored: %v", err)
	}
	if version != 0 || dirty {
		t.Errorf("version/dirty = %d/%v, want 0/false", version, dirty)
	}
}

func TestMigrateRequiresDSN(t *testing.T) {
	if err := Migrate("", os.DirFS("."), "migrations"); err == nil {
		t.Error("Migrate accepted an empty DSN")
	}
}

func TestMigrationDSNRewritesScheme(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"postgres://u:p@h:5432/db", "pgx5://u:p@h:5432/db"},
		{"postgresql://u:p@h:5432/db", "pgx5://u:p@h:5432/db"},
		{"pgx5://u:p@h:5432/db", "pgx5://u:p@h:5432/db"},
	}
	for _, tt := range tests {
		if got := migrationDSN(tt.in); got != tt.want {
			t.Errorf("migrationDSN(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
