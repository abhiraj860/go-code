// Package postgres provides connection pooling and schema migration for the
// services that own a Postgres database (catalog, inventory, order).
//
// Migrations run at service startup from an embedded filesystem rather than as
// a separate deploy step. That choice matters for the blue/green work in Phase
// 5: a new version must be able to boot against the existing schema without a
// human running anything first, and embedding the files means the binary and
// the schema it expects can never drift apart in a container registry.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options configures the connection pool.
type Options struct {
	// DSN is the libpq connection string. Required.
	DSN string

	// MaxConns bounds the pool. Sizing note: Postgres handles a few hundred
	// connections badly, and inventory-svc runs on N replicas -- N * MaxConns
	// must stay well under max_connections. Default 10.
	MaxConns int32

	// MinConns keeps warm connections so a traffic spike does not pay TCP plus
	// TLS handshake latency on every new request. Default 2.
	MinConns int32

	// MaxConnLifetime recycles connections so a long-lived pool eventually
	// rebalances after a failover. Default 1h.
	MaxConnLifetime time.Duration

	// ConnectTimeout bounds the initial dial. Default 5s.
	ConnectTimeout time.Duration
}

// Connect opens a pool and verifies it with a ping, so a bad DSN fails at
// startup rather than on the first query.
func Connect(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	if opts.DSN == "" {
		return nil, errors.New("postgres: DSN is required")
	}

	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: parsing DSN: %w", err)
	}

	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	} else {
		cfg.MaxConns = 10
	}
	if opts.MinConns > 0 {
		cfg.MinConns = opts.MinConns
	} else {
		cfg.MinConns = 2
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	} else {
		cfg.MaxConnLifetime = time.Hour
	}

	timeout := opts.ConnectTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(dialCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: creating pool: %w", err)
	}

	if err := pool.Ping(dialCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

// Migrate applies every pending migration from fsys, which is normally an
// embed.FS holding a service's migrations directory.
//
// It is safe to run concurrently from multiple replicas: golang-migrate takes a
// Postgres advisory lock for the duration, so during a rolling deploy the first
// replica migrates and the rest wait and then find nothing to do.
func Migrate(dsn string, fsys fs.FS, dir string) error {
	if dsn == "" {
		return errors.New("postgres: DSN is required for migration")
	}

	src, err := iofs.New(fsys, dir)
	if err != nil {
		return fmt.Errorf("postgres: opening migration source %q: %w", dir, err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrationDSN(dsn))
	if err != nil {
		return fmt.Errorf("postgres: initialising migrator: %w", err)
	}
	defer func() {
		// Close reports both a source and a database error; neither is
		// actionable once migration itself succeeded.
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: applying migrations: %w", err)
	}
	return nil
}

// Version reports the current schema version and whether the last migration
// left the database dirty. A dirty database means a migration failed partway
// and needs manual attention -- the service should refuse to serve traffic
// rather than run against a half-applied schema.
func Version(dsn string, fsys fs.FS, dir string) (version uint, dirty bool, err error) {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return 0, false, fmt.Errorf("postgres: opening migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrationDSN(dsn))
	if err != nil {
		return 0, false, fmt.Errorf("postgres: initialising migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil // no migrations applied yet
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: reading version: %w", err)
	}
	return version, dirty, nil
}

// migrationDSN adapts a standard postgres:// DSN to the scheme golang-migrate's
// pgx/v5 driver registers itself under.
func migrationDSN(dsn string) string {
	const (
		postgresPrefix   = "postgres://"
		postgresqlPrefix = "postgresql://"
		pgxPrefix        = "pgx5://"
	)
	switch {
	case len(dsn) > len(postgresqlPrefix) && dsn[:len(postgresqlPrefix)] == postgresqlPrefix:
		return pgxPrefix + dsn[len(postgresqlPrefix):]
	case len(dsn) > len(postgresPrefix) && dsn[:len(postgresPrefix)] == postgresPrefix:
		return pgxPrefix + dsn[len(postgresPrefix):]
	default:
		return dsn
	}
}

// ensure the pgx/v5 migrate driver is linked in; it registers itself via init.
var _ = pgx.Postgres{}
