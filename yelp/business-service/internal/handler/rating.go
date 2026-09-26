package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingHandler struct {
	DB *pgxpool.Pool
}

type UpdateRatingRequest struct {
	AvgRating   float64 `json:"avg_rating"`
	ReviewCount int     `json:"review_count"`
}

// UpdateRating is called only by the outbox worker to push a freshly
// recomputed average rating / review count onto a business row.
// Not exposed to end users - this is internal service-to-service traffic.
func (h *RatingHandler) UpdateRating(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cmdTag, err := h.DB.Exec(context.Background(), `
		UPDATE businesses
		SET avg_rating = $1, review_count = $2
		WHERE id = $3
	`, req.AvgRating, req.ReviewCount, id)
	if err != nil {
		http.Error(w, "failed to update rating", http.StatusInternalServerError)
		return
	}
	if cmdTag.RowsAffected() == 0 {
		http.Error(w, "business not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
