package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	APIPort string

	// Trie
	TopK            int
	RebuildInterval time.Duration

	// Data paths
	RawDataPath       string
	CleanedDataPath   string
	ProcessedDataPath string
	SnapshotPath      string

	// S3 (for later)
	S3Bucket string
	S3Region string
	S3Key    string
}

func Load() *Config {
	return &Config{
		// Server
		APIPort: getEnv("API_PORT", ":8080"),

		// Trie
		TopK:            getEnvInt("TOP_K", 5),
		RebuildInterval: getEnvDuration("REBUILD_INTERVAL", 24*time.Hour),

		// Data paths
		RawDataPath:       getEnv("RAW_DATA_PATH", "./data/raw-data.txt"),
		CleanedDataPath:   getEnv("CLEANED_DATA_PATH", "./data/cleaned-raw-data.txt"),
		ProcessedDataPath: getEnv("PROCESSED_DATA_PATH", "./data/aggregated-data.csv"),
		SnapshotPath:      getEnv("SNAPSHOT_PATH", "./data/trie-snapshot.bin"),

		// S3
		S3Bucket: getEnv("S3_BUCKET", "trie-snapshots"),
		S3Region: getEnv("S3_REGION", "us-east-1"),
		S3Key:    getEnv("S3_KEY", "trie-snapshot-latest.bin"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
