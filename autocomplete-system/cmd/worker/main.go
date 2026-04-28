package main

import (
	"autocomplete/internal/config"
	"autocomplete/internal/trie"
	"autocomplete/workers"
	"log"
	"time"
)

func main() {
	start := time.Now()
	log.Println("starting worker: aggregate data -> build new Trie -> serialize the Trie Snapshot -> upload to S3")
	cfg := config.Load()

	// 1. aggregate data
	log.Printf("step 1: aggregating data")
	// ok, err := workers.AggregateData(cfg.CleanedDataPath, cfg.ProcessedDataPath)
	// if !ok || err != nil {
	// 	log.Fatalf("aggregator failed: %v", err)
	// }
	log.Printf("step 1: done(skiped) | time=%s", time.Since(start))

	// 2. build new Trie
	log.Printf("step 2: building trie")
	trie := trie.NewAutoComplete(cfg.TopK)
	if err := trie.Build(cfg.ProcessedDataPath); err != nil {
		log.Fatalf("failed to construct the autocomplete system %v", err)
	}
	log.Printf("step 2: done | nodes=%d | time=%s", trie.GetNumNodes(), time.Since(start))

	// 3. serialize the Trie Snapshot
	log.Printf("step 3: serializing trie snapshot")
	if err := workers.SerializeTrie(trie, cfg.SnapshotPath); err != nil {
		log.Fatalf("failed to serialize trie: %v", err)
	}
	log.Printf("step 3: done | time=%s", time.Since(start))

	// 4. upload to S3
	log.Printf("step 4: uploading snapshot to S3")
	if err := workers.UploadSnapshot(cfg.SnapshotPath, cfg.S3Bucket, cfg.S3Key, cfg.S3Region); err != nil {
		log.Fatalf("failed to upload snapshot to S3: %v", err)
	}
	log.Printf("step 4: done | time=%s", time.Since(start))

	log.Printf("worker completed | time=%s", time.Since(start))
}
