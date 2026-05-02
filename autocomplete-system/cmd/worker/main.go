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
		log.Fatalf("failed to construct trie: %v", err)
	}
	log.Printf("step 2: done | time=%s | nodes=%d | trieSize=%d bytes ", time.Since(start), trie.GetNumNodes(), trie.TrieSize())

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

// func main() {
// 	ok, err := workers.ProcessTxtFile("./data/kaggle-dataset.txt")
// 	if !ok || err != nil {
// 		log.Fatalf("failed to process txt file: %v", err)
// 	}
// }

// 2026/04/30 17:45:43 starting worker: aggregate data -> build new Trie -> serialize the Trie Snapshot -> upload to S3
// 2026/04/30 17:45:43 step 1: aggregating data
// 2026/04/30 17:45:43 step 1: done(skiped) | time=1.087ms
// 2026/04/30 17:45:43 step 2: building trie
// 2026/04/30 17:46:01 step 2: done | time=16.5939245s | nodes=13739230 | trieSize=1674020878 bytes
// 2026/04/30 17:46:01 step 3: serializing trie snapshot
// 2026/04/30 17:46:17 step 3: done | time=34.2976829s
// 2026/04/30 17:46:17 step 4: uploading snapshot to S3
// 2026/04/30 17:47:17 snapshot uploaded to S3 | bucket=trie-snapshots | key=trie-snapshot-latest.bin | time=59.573401s
// 2026/04/30 17:47:17 step 4: done | time=1m33.8803746s
// 2026/04/30 17:47:17 worker completed | time=1m33.8803746s
