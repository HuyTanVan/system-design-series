package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/auth"
	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/model"
)

type ReviewHandler struct {
	DB *pgxpool.Pool
}

// GET businesses/:id/reviews?limit=10&offset=0
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	log.Printf("path: %s", r.URL.Path)
	log.Printf("route pattern: %s", chi.RouteContext(r.Context()).RoutePattern())
	log.Printf("params: %+v", chi.RouteContext(r.Context()).URLParams)
	businessID := chi.URLParam(r, "businessID")

	if businessID == "" {
		http.Error(w, "invalid business ID", http.StatusBadRequest)
		return
	}

	// limit := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 10
	}
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}
	fmt.Printf("%T\n", limit)
	rows, err := h.DB.Query(context.Background(), `
		SELECT reviews.id,
			reviews.business_id,
			reviews.user_id,
			users.name,
			reviews.rating, 
			reviews.text, 
			reviews.useful, 
			reviews.funny, 
			reviews.cool, 
			reviews.created_at
			FROM reviews
		JOIN users ON reviews.user_id = users.id
		WHERE business_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, businessID, limit, offset)
	if err != nil {
		http.Error(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reviews []model.Review
	for rows.Next() {
		var rv model.Review
		if err := rows.Scan(&rv.ID, &rv.BusinessID, &rv.UserID, &rv.UserName, &rv.Rating, &rv.Text, &rv.Useful, &rv.Funny, &rv.Cool, &rv.CreatedAt); err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		reviews = append(reviews, rv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// POST /businesses/:id/reviews
// request body: { "user_id": "...", "rating": 5, "text": "good food" }
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateReviewRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if req.BusinessID == "" || req.Rating < 1 || req.Rating > 5 {
		http.Error(
			w,
			"business_id required; rating must be 1-5",
			http.StatusBadRequest,
		)
		return
	}

	// 2. create a transaction to insert review and outbox record
	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "tx failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	reviewID := uuid.NewString()

	_, err = tx.Exec(ctx, `
		INSERT INTO reviews (id, business_id, user_id, rating, text, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, reviewID, req.BusinessID, userID, req.Rating, req.Text)
	if err != nil {
		http.Error(w, "insert review failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// same transaction: write to outbox so the worker can sync avg_rating to listings-service
	_, err = tx.Exec(ctx, `
		INSERT INTO review_outbox (business_id, rating)
		VALUES ($1, $2)
	`, req.BusinessID, req.Rating)
	if err != nil {
		http.Error(w, "insert outbox failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "commit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": reviewID})
}

// how to calculate average rating efficiently
// goal: whenever user searches for a business, display the average rating
// endpoint: user leaves a review
// approach 1: periodic update with background job
// 1. query review table
// 2. calculate average rating for each business
// 3. update a new average rating column in business table
// pros: when querying business, rating does along with it
// cons: not real-time

// approach 2: synchronous update + optimistic locking
//

// approach 3: outbox pattern(adding a table called review_outbox)
// 1. user submits a review, create a transaction
// 2. in transaction, insert review into reviews table
// 3. in transaction, insert a record into review_outbox table
// 4. a background worker processes the outbox where processed = false and updates the average rating in the business table
// pros:
// cons: a little latency due to the background processing, retries can happen
