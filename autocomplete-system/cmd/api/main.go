package main

import (
	"autocomplete/internal/api"
	"autocomplete/internal/config"
	"autocomplete/internal/event"
	"autocomplete/internal/kafka"
	"autocomplete/workers"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	// init producer
	producer := kafka.NewProducer(
		cfg.KafkaBootstrapServer,
		cfg.KafkaAPIKey,
		cfg.KafkaAPISecret,
		cfg.KafkaTopic,
	)
	defer producer.Close()

	// 1. download snapshot from S3
	log.Println("downloading snapshot from S3...")
	if err := workers.DownloadSnapshot(cfg.SnapshotPath, cfg.S3Bucket, cfg.S3Key, cfg.S3Region); err != nil {
		log.Fatalf("failed to download snapshot: %v", err)
	}
	log.Println("snapshot downloaded successfully")

	// 2. deserialize trie snapshot
	log.Println("deserializing trie snapshot...")
	ac, err := workers.DeserializeTrie(cfg.SnapshotPath, cfg.TopK)
	if err != nil {
		log.Fatalf("failed to deserialize trie: %v", err)
	}
	log.Printf("trie deserialized successfully | nodes=%d", ac.GetNumNodes())

	// 3. start background poller
	workers.StartPoller(ac, cfg.SnapshotPath, cfg.S3Bucket, cfg.S3Key, cfg.S3Region, cfg.RebuildInterval)
	log.Printf("poller started | interval=%s", cfg.RebuildInterval)

	// 4. init async event queue for logging search selections
	queue := event.NewAsyncEventQueue(producer, 10000)
	queue.Start() // run in background

	// 5. start API server
	r := gin.Default()
	h := api.NewHandler(ac)
	h.Producer = producer // assign the producer to the handler
	h.Queue = queue       // assign the event queue to the handler

	api.SetupRoutes(r, h)

	log.Printf("API server running on %s", cfg.APIPort)
	r.Run(cfg.APIPort)
}
