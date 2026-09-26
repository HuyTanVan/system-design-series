package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	batchSize    = 100
	pollInterval = 2 * time.Second
	maxAttempts  = 10
)

type outboxRow struct {
	ID         int64
	BusinessID string
	Operation  string
}

// Shape of the ES document. Adjust to your businesses table + ES mapping.
type businessDoc struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	City        string    `json:"city"`
	Location    *geoPoint `json:"location,omitempty"`
	AvgRating   float64   `json:"avg_rating"`
	ReviewCount int       `json:"review_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type geoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type worker struct {
	pool  *pgxpool.Pool
	esURL string
	index string
	http  *http.Client
}

func main() {
	dsn := getenv("BUSINESSES_DB_DSN", "postgres://yelp:yelp@localhost:5432/businesses_db")
	esURL := getenv("ELASTICSEARCH_URL", "http://elasticsearch:9200")
	index := getenv("ES_INDEX", "businesses")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to businesses db: %v", err)
	}
	defer pool.Close()

	w := &worker{
		pool:  pool,
		esURL: esURL,
		index: index,
		http:  &http.Client{Timeout: 10 * time.Second},
	}

	log.Printf("es sync worker started, polling every %s", pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			// drain: keep going while there is a full batch waiting
			for {
				n, err := w.processBatch(ctx)
				if err != nil {
					log.Printf("batch error: %v", err)
					break
				}
				if n < batchSize {
					break
				}
			}
		}
	}
}

// processBatch claims rows inside a transaction with SKIP LOCKED, so multiple
// workers can run concurrently without processing the same rows.
// Returns the number of outbox rows claimed.
func (w *worker) processBatch(ctx context.Context) (int, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) // no-op after commit

	rows, err := tx.Query(ctx, `
		SELECT id, business_id, operation
		FROM business_outbox
		WHERE processed = false AND attempts < $1
		ORDER BY id
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, maxAttempts, batchSize)
	if err != nil {
		return 0, err
	}

	var batch []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.ID, &r.BusinessID, &r.Operation); err != nil {
			rows.Close()
			return 0, err
		}
		batch = append(batch, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(batch) == 0 {
		return 0, nil
	}

	log.Printf("processing %d outbox rows", len(batch))

	// Dedupe: only the latest state matters, so sync each business once.
	grouped := make(map[string][]int64)
	for _, r := range batch {
		grouped[r.BusinessID] = append(grouped[r.BusinessID], r.ID)
	}

	var okIDs, failedIDs []int64
	for businessID, ids := range grouped {
		if err := w.syncBusiness(ctx, tx, businessID); err != nil {
			log.Printf("failed to sync business %s (%d rows): %v", businessID, len(ids), err)
			failedIDs = append(failedIDs, ids...)
			continue
		}
		okIDs = append(okIDs, ids...)
	}

	if len(okIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE business_outbox SET processed = true WHERE id = ANY($1)`, okIDs); err != nil {
			return 0, err
		}
	}
	if len(failedIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE business_outbox SET attempts = attempts + 1 WHERE id = ANY($1)`, failedIDs); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(batch), nil
}

// syncBusiness reads the CURRENT state of the row and makes ES match it.
// It ignores the operation type on purpose: if the row exists, index it;
// if it doesn't, delete it. That makes it immune to duplicate/out-of-order events.
func (w *worker) syncBusiness(ctx context.Context, tx pgx.Tx, businessID string) error {
	var doc businessDoc
	var lat, lon *float64

	err := tx.QueryRow(ctx, `
		SELECT id, name, COALESCE(description, ''), COALESCE(category, ''),
		       COALESCE(city, ''), latitude, longitude,
		       COALESCE(avg_rating, 0), COALESCE(review_count, 0), updated_at
		FROM businesses
		WHERE id = $1
	`, businessID).Scan(
		&doc.ID, &doc.Name, &doc.Description, &doc.Category,
		&doc.City, &lat, &lon,
		&doc.AvgRating, &doc.ReviewCount, &doc.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return w.esDelete(ctx, businessID)
	}
	if err != nil {
		return err
	}
	if lat != nil && lon != nil {
		doc.Location = &geoPoint{Lat: *lat, Lon: *lon}
	}
	return w.esIndex(ctx, doc)
}

func (w *worker) esIndex(ctx context.Context, doc businessDoc) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/_doc/%s", w.esURL, w.index, doc.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return w.do(req, false)
}

func (w *worker) esDelete(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", w.esURL, w.index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	return w.do(req, true)
}

func (w *worker) do(req *http.Request, allow404 bool) error {
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && allow404 {
		return nil // already gone, delete is idempotent
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("elasticsearch returned %d: %s", resp.StatusCode, b)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
