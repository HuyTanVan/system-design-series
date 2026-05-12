package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port    string // HTTP listen port, e.g. "8080"
	BaseURL string // Public base URL, e.g. "https://sho.rt" — used to build short links

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	DefaultTTLDays int // how many days a link lives before expiry (0 = forever)
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}

	// Required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	// Optional int fields with defaults
	var err error
	cfg.RedisDB, err = getEnvInt("REDIS_DB", 0)
	if err != nil {
		return nil, fmt.Errorf("config: REDIS_DB must be an integer: %w", err)
	}

	cfg.DefaultTTLDays, err = getEnvInt("DEFAULT_TTL_DAYS", 0)
	if err != nil {
		return nil, fmt.Errorf("config: DEFAULT_TTL_DAYS must be an integer: %w", err)
	}

	return cfg, nil
}

// getEnv returns the value of key, or fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}
