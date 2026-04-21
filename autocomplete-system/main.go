package main

import (
	"autocomplete/config"
	"autocomplete/logger"
	"autocomplete/trie"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func searchHandler(ac *trie.AutoComplete) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, "missing q", http.StatusBadRequest)
			return
		}

		results := ac.Search(q)
		// fmt.Println(values)
		// results := make([]string, len(values))
		// for i, v := range values {
		// 	results[i] = v.Text
		// }

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
func queryHandler(logger *logger.QueryLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		if body.Query == "" {
			http.Error(w, "missing query", http.StatusBadRequest)
			return
		}

		if err := logger.Log(body.Query); err != nil {
			http.Error(w, "failed to log", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
func sizeHandler(ac *trie.AutoComplete) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		size := ac.Size()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"size": size})
	}
}
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		start := time.Now()

		next.ServeHTTP(w, r)

		if r.URL.RawQuery != "" {
			log.Printf("%s %s?%s — %s", r.Method, r.URL.Path, r.URL.RawQuery, time.Since(start))
		} else {
			log.Printf("%s %s — %s", r.Method, r.URL.Path, time.Since(start))
		}
	})
}

// Starts a background worker that rebuilds the trie periodically,
// applied retry logic with exponential backoff to handle potential build failures gracefully.
// If rebuilding fails after all retires, keep the old trie -> no downtime.
func startRebuildWorker(ac *trie.AutoComplete, cfg config.Config) {
	go func() {
		ticker := time.NewTicker(cfg.RebuildInterval)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("rebuild worker: starting...")
			start := time.Now()

			var newAc *trie.AutoComplete
			var err error

			// retry 3 times
			for attempt := 1; attempt <= 3; attempt++ {
				newAc = trie.NewAutoComplete(cfg.TopK)
				err = newAc.Build(currentLogPath(cfg.LogDir))
				if err == nil {
					break
				}
				log.Printf("rebuild worker: attempt %d failed: %v", attempt, err)
				time.Sleep(time.Duration(attempt) * time.Second) // backoff: 1s, 2s, 3s
			}

			if err != nil {
				log.Printf("rebuild worker: all retries failed, keeping old trie")
				continue
			}

			ac.Swap(newAc)

			log.Printf("rebuild worker: done | time=%s | size=%d", time.Since(start), ac.Size())
		}
	}()
}
func currentLogPath(dir string) string {
	date := time.Now().Format("2006-01-02")
	return filepath.Join(dir, fmt.Sprintf("query-log-%s.txt", date))
}
func main() {
	// 1. load configuration
	cfg := config.Config{
		TopK:            5,
		RebuildInterval: time.Second * 10,
		LogDir:          "./logger/logs",
	}
	newCfg := config.NewConfig(cfg)

	// 2. init logger
	queryLogger, err := logger.NewQueryLogger(currentLogPath(cfg.LogDir))
	if err != nil {
		log.Fatalf("failed to init a logger: %v", err)
	}

	// 3. pre-build autocomplete system
	start := time.Now()
	ac := trie.NewAutoComplete(newCfg.TopK)
	if err := ac.Build(currentLogPath(cfg.LogDir)); err != nil {
		log.Fatalf("failed to construct the autocomplete system %v", err)
	}
	log.Printf("autocomplete system built sucessfully | time=%s | size=%d", time.Since(start), ac.Size())

	// 4. start background worker to rebuild trie periodically
	startRebuildWorker(ac, newCfg)

	// 5. start http server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})
	mux.HandleFunc("/search", searchHandler(ac))
	mux.HandleFunc("/query", queryHandler(queryLogger))
	mux.HandleFunc("/size", sizeHandler(ac))

	log.Println("server running on :8080")
	http.ListenAndServe(":8080", loggingMiddleware(mux))

}
