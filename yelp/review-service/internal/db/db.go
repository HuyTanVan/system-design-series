package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string, minConns, maxConns int) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	config.MinConns = int32(minConns)
	config.MaxConns = int32(maxConns)
	config.MaxConnIdleTime = 0 // or a long duration, so idle conns aren't reaped

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := pool.Stat()

			fmt.Printf(
				"DB Pool: total=%d acquired=%d idle=%d empty_acquire=%d wait=%v\n",
				stats.TotalConns(),
				stats.AcquiredConns(),
				stats.IdleConns(),
				stats.EmptyAcquireCount(),
				stats.EmptyAcquireWaitTime(),
			)
		}
		// 	total        = how many connections exist
		// acquired     = how many connections are currently being used
		// idle         = how many connections are available
		// empty_acquire = how many times someone tried to get a connection when none was available
		// wait         = total time those requests spent waiting for a connection
	}()
	return pool, nil
}
