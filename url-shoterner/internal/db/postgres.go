package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool.Pool so the rest of the app never imports pgx directly.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates and validates a PostgreSQL connection pool.
// It retries up to 5 times with exponential backoff — useful on Railway
// where the DB container may not be ready the instant the app starts.
func Connect(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: invalid DSN: %w", err)
	}

	// Pool tuning — sensible defaults for a small Railway instance.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	var pool *pgxpool.Pool
	backoff := time.Second

	for attempt := 1; attempt <= 5; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			} else {
				pool.Close()
				err = pingErr
			}
		}

		if attempt == 5 {
			return nil, fmt.Errorf("db: could not connect after %d attempts: %w", attempt, err)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("db: context cancelled while connecting: %w", ctx.Err())
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return &DB{Pool: pool}, nil
}

// Close shuts down the connection pool gracefully.
func (d *DB) Close() {
	d.Pool.Close()
}

// Ping checks that the database is still reachable.
func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}
