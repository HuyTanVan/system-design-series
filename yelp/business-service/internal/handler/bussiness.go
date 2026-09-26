package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/HuyTanVan/yelp-clone/internal/model"
)

type BusinessHandler struct {
	DB *pgxpool.Pool
}

func (h *BusinessHandler) TestGetByID(w http.ResponseWriter, r *http.Request) {
	b := model.Business{
		ID: "OklxXDBntbfM956ZYtD5IA",
	}
	query := `SELECT id FROM businesses WHERE id = 'OklxXDBntbfM956ZYtD5IA'`
	err := h.DB.QueryRow(context.Background(), query).Scan(new(string))
	if err != nil {
		http.Error(w, "business not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// view business's details
func (h *BusinessHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "invalid business ID", http.StatusBadRequest)
		return
	}

	var b model.Business

	// TODO: add attributes and hours to be returned in the query
	query := `
		SELECT id, name, address, city, state, postal_code,
		       ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
		       avg_rating, review_count, is_open
		FROM businesses
		WHERE id = $1
	`
	err := h.DB.QueryRow(context.Background(), query, id).Scan(
		&b.ID, &b.Name, &b.Address, &b.City, &b.State, &b.PostalCode,
		&b.Lat, &b.Lng, &b.AvgRating, &b.ReviewCount, &b.IsOpen,
	)
	if err != nil {
		http.Error(w, "business not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}
