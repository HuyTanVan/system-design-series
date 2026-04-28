package main

import (
	"autocomplete/internal/api"
	"autocomplete/internal/config"
	"autocomplete/workers"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// 1. download snapshot from S3
	log.Println("downloading snapshot from S3...")
	if err := workers.DownloadSnapshot(cfg.SnapshotPath, cfg.S3Bucket, cfg.S3Key, cfg.S3Region); err != nil {
		log.Fatalf("failed to download snapshot: %v", err)
	}

	// 2. deserialize trie
	ac, err := workers.DeserializeTrie(cfg.SnapshotPath, cfg.TopK)
	if err != nil {
		log.Fatalf("failed to deserialize trie: %v", err)
	}
	log.Printf("trie loaded | nodes=%d", ac.Size())

	// 3. start background poller
	workers.StartPoller(ac, cfg.SnapshotPath, cfg.S3Bucket, cfg.S3Key, cfg.S3Region, cfg.RebuildInterval)
	log.Printf("poller started | interval=%s", cfg.RebuildInterval)

	// 4. setup and serve
	r := gin.Default()
	h := api.NewHandler(ac)
	api.SetupRoutes(r, h)

	log.Printf("API server running on %s", cfg.APIPort)
	r.Run(":", cfg.APIPort)
}
