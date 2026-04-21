package config

import "time"

type Config struct {
	TopK            int           // how many suggestions to return for each query
	RebuildInterval time.Duration // how often to rebuild the trie, e.g., every day or every week
	LogDir          string        // where to store the query logs
}

// default configuration values
func NewConfig(cfg Config) Config {

	return Config{
		TopK:            cfg.TopK,
		RebuildInterval: cfg.RebuildInterval,
		LogDir:          cfg.LogDir,
	}
}
