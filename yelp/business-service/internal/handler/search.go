// package handler

// import (
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"strconv"

// 	"github.com/jackc/pgx/v5/pgxpool"

// 	"github.com/HuyTanVan/yelp-clone/internal/model"
// )

// type SearchHandler struct {
// 	DB *pgxpool.Pool
// }

// // GET /businesses/search?lat=34.42&lng=-119.71&radius_m=5000&category=Restaurants&limit=20
// func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
// 	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
// 	if err != nil {
// 		http.Error(w, "invalid lat", http.StatusBadRequest)
// 		return
// 	}
// 	lng, err := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
// 	if err != nil {
// 		http.Error(w, "invalid lng", http.StatusBadRequest)
// 		return
// 	}

// 	radiusM := 5000.0
// 	if v := r.URL.Query().Get("radius_m"); v != "" {
// 		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
// 			radiusM = parsed
// 		}
// 	}

// 	limit := 10
// 	if v := r.URL.Query().Get("limit"); v != "" {
// 		if parsed, err := strconv.Atoi(v); err == nil {
// 			limit = parsed
// 		}
// 	}

// 	category := r.URL.Query().Get("category")

// 	query := `
// 		SELECT b.id, b.name, b.address, b.city, b.state, b.postal_code,
// 		       ST_Y(b.location::geometry) AS lat, ST_X(b.location::geometry) AS lng,
// 		       b.avg_rating, b.review_count, b.is_open,
// 		       ST_Distance(b.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m
// 		FROM businesses b
// 	`
// 	args := []interface{}{lng, lat}
// 	argN := 3

// 	if category != "" {
// 		query += `
// 			JOIN business_categories bc ON bc.business_id = b.id
// 			JOIN categories c ON c.id = bc.category_id AND c.name = $` + strconv.Itoa(argN) + `
// 		`
// 		args = append(args, category)
// 		argN++
// 	}

// 	query += `
// 		WHERE ST_DWithin(b.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $` + strconv.Itoa(argN) + `)
// 		ORDER BY distance_m
// 		LIMIT $` + strconv.Itoa(argN+1)
// 	args = append(args, radiusM, limit)

// 	rows, err := h.DB.Query(context.Background(), query, args...)
// 	if err != nil {
// 		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	defer rows.Close()

// 	type result struct {
// 		model.Business
// 		DistanceM float64 `json:"distance_m"`
// 	}

// 	var results []result
// 	for rows.Next() {
// 		var res result
// 		err := rows.Scan(
// 			&res.ID, &res.Name, &res.Address, &res.City, &res.State, &res.PostalCode,
// 			&res.Lat, &res.Lng, &res.AvgRating, &res.ReviewCount, &res.IsOpen,
// 			&res.DistanceM,
// 		)
// 		if err != nil {
// 			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 		results = append(results, res)
// 	}

//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(results)
//	}
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/HuyTanVan/yelp-clone/internal/model"
)

type SearchHandler struct {
	DB *pgxpool.Pool
}

// GET /businesses/search?lat=34.42&lng=-119.71&radius_m=5000&category=Restaurants&limit=20
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		http.Error(w, "invalid lat", http.StatusBadRequest)
		return
	}
	lng, err := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err != nil {
		http.Error(w, "invalid lng", http.StatusBadRequest)
		return
	}

	radiusM := 5000.0
	if v := r.URL.Query().Get("radius_m"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			radiusM = parsed
		}
	}

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}

	category := r.URL.Query().Get("category")

	query := `
		SELECT b.id, b.name, b.address, b.city, b.state, b.postal_code,
		       ST_Y(b.location::geometry) AS lat, ST_X(b.location::geometry) AS lng,
		       b.avg_rating, b.review_count, b.is_open,
		       ST_Distance(b.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m
		FROM businesses b
		WHERE ST_DWithin(b.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
	`
	args := []interface{}{lng, lat, radiusM}
	argN := 4

	if category != "" {
		query += `
			AND EXISTS (
				SELECT 1 FROM business_categories bc
				JOIN categories c ON c.id = bc.category_id AND c.name = $` + strconv.Itoa(argN) + `
				WHERE bc.business_id = b.id
			)
		`
		args = append(args, category)
		argN++
	}

	query += `
		ORDER BY distance_m
		LIMIT $` + strconv.Itoa(argN)
	args = append(args, limit)

	rows, err := h.DB.Query(context.Background(), query, args...)
	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type result struct {
		model.Business
		DistanceM float64 `json:"distance_m"`
	}

	var results []result
	for rows.Next() {
		var res result
		err := rows.Scan(
			&res.ID, &res.Name, &res.Address, &res.City, &res.State, &res.PostalCode,
			&res.Lat, &res.Lng, &res.AvgRating, &res.ReviewCount, &res.IsOpen,
			&res.DistanceM,
		)
		if err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, res)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
