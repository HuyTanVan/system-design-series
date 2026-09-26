// Streams business.json line by line (NDJSON) and inserts into businesses_db.
// Does NOT load the whole file into memory — reads one line at a time.
//
// Usage:
//
//	go run cmd/loader/main.go /path/to/business.json
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type yelpBusiness struct {
	BusinessID  string  `json:"business_id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	PostalCode  string  `json:"postal_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Stars       float64 `json:"stars"`
	ReviewCount int     `json:"review_count"`
	IsOpen      int     `json:"is_open"`
	Categories  string  `json:"categories"` // comma-separated, e.g. "Doctors, Health & Medical"
}

func main() {
	start := time.Now()
	if len(os.Args) < 2 {
		log.Fatal("usage: go run cmd/loader/main.go /path/to/business.json")
	}
	path := os.Args[1]

	dsn := os.Getenv("BUSINESSES_DB_DSN")
	if dsn == "" {
		dsn = "postgres://yelp:yelp@localhost:5432/businesses_db"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // handle long lines

	categoryCache := make(map[string]int) // name -> id, avoid re-querying every row

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var b yelpBusiness
		if err := json.Unmarshal([]byte(line), &b); err != nil {
			log.Printf("skip: bad json: %v", err)
			continue
		}

		if err := insertBusiness(ctx, pool, b, categoryCache); err != nil {
			log.Printf("skip %s: %v", b.BusinessID, err)
			continue
		}

		count++
		if count%1000 == 0 {
			log.Printf("loaded %d businesses...", count)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scan error: %v", err)
	}

	log.Printf("done. loaded: %d businesses total. time taken: %v", count, time.Since(start))
}

func insertBusiness(ctx context.Context, pool *pgxpool.Pool, b yelpBusiness, categoryCache map[string]int) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	_, err = tx.Exec(ctx, `
		INSERT INTO businesses (id, name, address, city, state, postal_code, location, avg_rating, review_count, is_open)
		VALUES ($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`,
		b.BusinessID, b.Name, b.Address, b.City, b.State, b.PostalCode,
		b.Longitude, b.Latitude, // note: lng first, then lat
		b.Stars, b.ReviewCount, b.IsOpen == 1,
	)
	if err != nil {
		return fmt.Errorf("insert business: %w", err)
	}

	// split "Doctors, Health & Medical, Acupuncture" -> individual categories
	if b.Categories != "" {
		for _, raw := range strings.Split(b.Categories, ",") {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}

			catID, ok := categoryCache[name]
			if !ok {
				err := tx.QueryRow(ctx, `
					INSERT INTO categories (name) VALUES ($1)
					ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
					RETURNING id
				`, name).Scan(&catID)
				if err != nil {
					return fmt.Errorf("upsert category %q: %w", name, err)
				}
				categoryCache[name] = catID
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO business_categories (business_id, category_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, b.BusinessID, catID)
			if err != nil {
				return fmt.Errorf("link category %q: %w", name, err)
			}
		}
	}

	return tx.Commit(ctx)
}
