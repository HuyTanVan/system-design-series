package workers

import (
	"autocomplete/internal/trie"
	"log"
	"time"
)

func StartPoller(ac *trie.AutoComplete, snapshotPath, bucket, key, region string, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			log.Println("poller: checking for new snapshot...")

			// 1. download new snapshot
			if err := DownloadSnapshot(snapshotPath, bucket, key, region); err != nil {
				log.Printf("poller: failed to download snapshot: %v", err)
				continue
			}

			// 2. deserialize into new trie
			newAc, err := DeserializeTrie(snapshotPath, ac.K())
			if err != nil {
				log.Printf("poller: failed to deserialize snapshot: %v", err)
				continue
			}

			// 3. swap old trie with new trie
			ac.Swap(newAc)
			log.Printf("poller: trie swapped successfully | nodes=%d", newAc.Size())
		}
	}()
}
