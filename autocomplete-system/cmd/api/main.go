package main

import (
	"autocomplete/internal/api"
	"autocomplete/internal/config"
	"autocomplete/internal/events"
	"autocomplete/internal/kafka"
	"autocomplete/workers"
	"fmt"
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
	fmt.Printf("Kafka bootstrapServer = %q\n", cfg.KafkaBootstrapServer)
	defer producer.Close()

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
	bus := events.NewEventBus(producer, 10000)
	bus.Start()

	r := gin.Default()
	h := api.NewHandler(ac)
	h.Producer = producer // assign the producer to the handler
	h.Bus = bus           // assign the event bus to the handler

	api.SetupRoutes(r, h)

	log.Printf("API server running on %s", cfg.APIPort)
	r.Run(cfg.APIPort)
}
