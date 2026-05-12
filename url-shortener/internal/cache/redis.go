package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a redis.Client and exposes only the operations the app needs.
// Keeping a thin wrapper means we can swap the underlying client in tests.
type Client struct {
	rdb *redis.Client
}

// Connect creates and validates a Redis connection.
// It retries up to 5 times with exponential backoff, matching the DB package.
func Connect(ctx context.Context, addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	var err error
	backoff := time.Second

	for attempt := 1; attempt <= 5; attempt++ {
		if err = rdb.Ping(ctx).Err(); err == nil {
			break
		}

		if attempt == 5 {
			_ = rdb.Close()
			return nil, fmt.Errorf("cache: could not connect after %d attempts: %w", attempt, err)
		}

		select {
		case <-ctx.Done():
			_ = rdb.Close()
			return nil, fmt.Errorf("cache: context cancelled while connecting: %w", ctx.Err())
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return &Client{rdb: rdb}, nil
}

// Close shuts down the Redis connection pool.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping checks that Redis is still reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// ── Cache-aside helpers ───────────────────────────────────────────────────────

// Get retrieves a string value. Returns ("", nil) on a cache miss so callers
// can distinguish between "not found" and a real error.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // cache miss — not an error
	}
	return val, err
}

// Set stores a key-value pair with an optional TTL (0 = no expiry).
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Delete removes one or more keys. Silently ignores missing keys.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// ── Atomic counter (Base62 source) ───────────────────────────────────────────

// Increment atomically increments a counter key and returns the new value.
// This is the Redis-native replacement for Postgres BIGSERIAL — the service
// layer calls Increment("counter:url") then passes the result to base62.Encode.
func (c *Client) Increment(ctx context.Context, key string) (uint64, error) {
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: increment %q: %w", key, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("cache: counter %q overflowed into negative", key)
	}
	return uint64(n), nil
}
