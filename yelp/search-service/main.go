package main

import (
	"context"
	"log"
	"os"

	"github.com/HuyTanVan/yelp-clone/search-service/internal/es"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	businessesDSN := os.Getenv("BUSINESSES_DB_DSN")
	if businessesDSN == "" {
		businessesDSN = "postgres://yelp:yelp@localhost:5432/businesses_db"
	}
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	pool, err := pgxpool.New(ctx, businessesDSN)
	if err != nil {
		log.Fatalf("failed to connect to businesses db: %v", err)
	}
	defer pool.Close()

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{esURL}})
	if err != nil {
		log.Fatalf("failed to create es client: %v", err)
	}

	if err := es.EnsureIndex(ctx, esClient); err != nil {
		log.Fatalf("failed to ensure index: %v", err)
	}

	// pull businesses along with their categories, aggregated into an array
	rows, err := pool.Query(ctx, `
		SELECT b.id, b.name, b.city, b.state, b.avg_rating, b.review_count,
		       COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') AS categories
		FROM businesses b
		LEFT JOIN business_categories bc ON bc.business_id = b.id
		LEFT JOIN categories c ON c.id = bc.category_id
		GROUP BY b.id, b.name, b.city, b.state, b.avg_rating, b.review_count
	`)
	if err != nil {
		log.Fatalf("failed to query businesses: %v", err)
	}
	defer rows.Close()

	count := 0
	failed := 0
	for rows.Next() {
		var doc es.BusinessDoc
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.City, &doc.State, &doc.AvgRating, &doc.ReviewCount, &doc.Categories); err != nil {
			log.Printf("failed to scan row: %v", err)
			failed++
			continue
		}

		if err := es.IndexBusiness(ctx, esClient, doc); err != nil {
			log.Printf("failed to index business %s: %v", doc.ID, err)
			failed++
			continue
		}

		count++
		if count%5000 == 0 {
			log.Printf("indexed %d businesses so far...", count)
		}
	}

	log.Printf("backfill complete: %d indexed, %d failed", count, failed)
}
