package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRow struct {
	ID         string
	BusinessID string
}

type updateRatingRequest struct {
	AvgRating   float64 `json:"avg_rating"`
	ReviewCount int     `json:"review_count"`
}

func main() {
	reviewsDSN := os.Getenv("REVIEWS_DB_DSN")
	if reviewsDSN == "" {
		reviewsDSN = "postgres://yelp:yelp@localhost:5432/reviews_db"
	}
	businessesServiceURL := os.Getenv("BUSINESSES_SERVICE_URL")
	if businessesServiceURL == "" {
		businessesServiceURL = "http://business-service:8081"
	}

	pool, err := db.NewPool(context.Background(), reviewsDSN, 1, 4)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	log.Println("outbox worker started, polling every 5s")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		processBatch(pool, businessesServiceURL)
	}
}

// processBatch: queries the outbox table for unprocessed rows, groups them by business_id,
// and recomputes the average rating for each business.
func processBatch(pool *pgxpool.Pool, businessesServiceURL string) {
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT id, business_id
		FROM review_outbox
		WHERE processed = false
		ORDER BY id
		LIMIT 50
	`)
	if err != nil {
		log.Printf("failed to query outbox: %v", err)
		return
	}

	var batch []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.ID, &row.BusinessID); err != nil {
			log.Printf("failed to scan outbox row: %v", err)
			continue
		}
		batch = append(batch, row)
	}
	rows.Close()

	if len(batch) == 0 {
		return // nothing to do this cycle
	}

	log.Printf("processing %d outbox rows", len(batch))

	// group by business_id - if one business got 100 reviews in this window,
	// we only need to recompute its average ONCE, not 100 times.
	// every row in the group still gets marked processed, since the single
	// recompute already reflects all of them (it queries all reviews for
	// that business, not just the one that triggered this row).
	grouped := make(map[string][]outboxRow)
	// grouped: b_id -> [row1, row2, row3]
	for _, row := range batch {
		grouped[row.BusinessID] = append(grouped[row.BusinessID], row)
	}

	for businessID, group := range grouped {
		if err := processBusiness(ctx, pool, businessesServiceURL, businessID, group); err != nil {
			log.Printf("failed to process business %s (%d outbox rows): %v", businessID, len(group), err)
			// all rows in this group stay processed=false, retried next cycle
			continue
		}
	}
}

func processBusiness(
	ctx context.Context,
	pool *pgxpool.Pool,
	businessesServiceURL string,
	businessID string,
	group []outboxRow) error {

	// recompute the average from scratch - simple, always correct, and covers
	// every review currently in the table, not just the ones in this group
	var avgRating float64
	var reviewCount int
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM reviews
		WHERE business_id = $1
	`, businessID).Scan(&avgRating, &reviewCount)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(updateRatingRequest{
		AvgRating:   avgRating,
		ReviewCount: reviewCount,
	})
	url := businessesServiceURL + "/businesses/" + businessID + "/rating"
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("businesses-service returned status %d", resp.StatusCode)
	}

	// mark every outbox row in this group as processed - one recompute covers all of them
	ids := make([]string, len(group))
	for i, row := range group {
		ids[i] = row.ID
	}
	_, err = pool.Exec(ctx, `UPDATE review_outbox SET processed = true WHERE id = ANY($1)`, ids)
	return err
}
