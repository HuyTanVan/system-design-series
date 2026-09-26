// Streams user.json and review.json (NDJSON) into reviews_db, one line at a time.
//
// Usage:
//
//	go run cmd/loader/main.go /path/to/user.json /path/to/review.json
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type yelpUser struct {
	UserID       string  `json:"user_id"`
	Name         string  `json:"name"`
	ReviewCount  int     `json:"review_count"`
	AverageStars float64 `json:"average_stars"`
}

type yelpReview struct {
	ReviewID   string  `json:"review_id"`
	UserID     string  `json:"user_id"`
	BusinessID string  `json:"business_id"`
	Stars      float64 `json:"stars"`
	Text       string  `json:"text"`
	Useful     int     `json:"useful"`
	Funny      int     `json:"funny"`
	Cool       int     `json:"cool"`
	Date       string  `json:"date"` // "2018-07-07 22:09:11"
}

const dateLayout = "2006-01-02 15:04:05"

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: go run cmd/loader/main.go /path/to/user.json /path/to/review.json")
	}
	_ = os.Args[1]
	reviewPath := os.Args[2]

	dsn := os.Getenv("REVIEWS_DB_DSN")
	if dsn == "" {
		dsn = "postgres://yelp:yelp@localhost:5432/reviews_db"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// if err := loadUsers(ctx, pool, userPath); err != nil {
	// 	log.Fatalf("load users: %v", err)
	// }

	if err := loadReviews(ctx, pool, reviewPath); err != nil {
		log.Fatalf("load reviews: %v", err)
	}
}

func loadUsers(ctx context.Context, pool *pgxpool.Pool, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	const batchSize = 5000
	batch := make([][]interface{}, 0, batchSize)
	total := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire conn: %w", err)
		}
		defer conn.Release()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, `
			CREATE TEMP TABLE tmp_users (
				id TEXT, name TEXT, review_count INT, average_stars NUMERIC(3,2)
			) ON COMMIT DROP
		`)
		if err != nil {
			return fmt.Errorf("create temp table: %w", err)
		}

		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"tmp_users"},
			[]string{"id", "name", "review_count", "average_stars"},
			pgx.CopyFromRows(batch),
		)
		if err != nil {
			return fmt.Errorf("copy users: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, name, review_count, average_stars)
			SELECT id, name, review_count, average_stars FROM tmp_users
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return fmt.Errorf("insert from temp: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		total += len(batch)
		batch = batch[:0]
		log.Printf("loaded %d users...", total)
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var u yelpUser
		if err := json.Unmarshal([]byte(line), &u); err != nil {
			log.Printf("skip user: bad json: %v", err)
			continue
		}

		batch = append(batch, []interface{}{u.UserID, u.Name, u.ReviewCount, u.AverageStars})
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}

	log.Printf("done loading users. total: %d", total)
	return scanner.Err()
}

func loadReviews(ctx context.Context, pool *pgxpool.Pool, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	const batchSize = 5000
	batch := make([][]interface{}, 0, batchSize)
	total := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire conn: %w", err)
		}
		defer conn.Release()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, `
			CREATE TEMP TABLE tmp_reviews (
				id TEXT, business_id TEXT, user_id TEXT, rating SMALLINT,
				text TEXT, useful INT, funny INT, cool INT, created_at TIMESTAMPTZ
			) ON COMMIT DROP
		`)
		if err != nil {
			return fmt.Errorf("create temp table: %w", err)
		}

		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"tmp_reviews"},
			[]string{"id", "business_id", "user_id", "rating", "text", "useful", "funny", "cool", "created_at"},
			pgx.CopyFromRows(batch),
		)
		if err != nil {
			return fmt.Errorf("copy reviews: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO reviews (id, business_id, user_id, rating, text, useful, funny, cool, created_at)
			SELECT id, business_id, user_id, rating, text, useful, funny, cool, created_at FROM tmp_reviews
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return fmt.Errorf("insert from temp: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		total += len(batch)
		batch = batch[:0]
		log.Printf("loaded %d reviews...", total)
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var r yelpReview
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			log.Printf("skip review: bad json: %v", err)
			continue
		}

		createdAt, err := time.Parse(dateLayout, r.Date)
		if err != nil {
			log.Printf("skip review %s: bad date: %v", r.ReviewID, err)
			continue
		}

		batch = append(batch, []interface{}{
			r.ReviewID, r.BusinessID, r.UserID, int(r.Stars), r.Text, r.Useful, r.Funny, r.Cool, createdAt,
		})
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}

	log.Printf("done loading reviews. total: %d", total)
	return scanner.Err()
}
