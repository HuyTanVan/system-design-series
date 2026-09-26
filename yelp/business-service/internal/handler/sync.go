package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessToESHandler struct {
	DB *pgxpool.Pool
}

type createBusinessRequest struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Address    string  `json:"address"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	PostalCode string  `json:"postal_code"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
}

type updateBusinessRequest struct {
	Name       *string  `json:"name,omitempty"`
	Address    *string  `json:"address,omitempty"`
	City       *string  `json:"city,omitempty"`
	State      *string  `json:"state,omitempty"`
	PostalCode *string  `json:"postal_code,omitempty"`
	Lat        *float64 `json:"lat,omitempty"`
	Lon        *float64 `json:"lon,omitempty"`
	IsOpen     *bool    `json:"is_open,omitempty"`
}

// CreateBusiness inserts a row into businesses. The AFTER INSERT trigger
// writes a business_outbox row; this handler does not touch ES or the
// outbox table directly.
func (h *BusinessToESHandler) CreateBusiness(w http.ResponseWriter, r *http.Request) {
	var req createBusinessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.Name == "" {
		http.Error(w, "id and name are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	_, err := h.DB.Exec(ctx, `
		INSERT INTO businesses (id, name, address, city, state, postal_code, location)
		VALUES ($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography)
	`, req.ID, req.Name, req.Address, req.City, req.State, req.PostalCode, req.Lon, req.Lat)
	// Note: ST_MakePoint takes (lon, lat), not (lat, lon) - easy to swap by mistake.

	if err != nil {
		http.Error(w, "failed to insert business: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": req.ID, "status": "created"})
}

// UpdateBusiness does a partial update, only setting columns present in the
// request body. The AFTER UPDATE trigger writes a business_outbox row if the
// row actually changed.
func (h *BusinessToESHandler) UpdateBusiness(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var req updateBusinessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// COALESCE keeps existing value when the field wasn't sent.
	// Lat/lon are handled as a pair: if either is provided, both must be,
	// since ST_MakePoint needs both to build a point.
	var lat, lon *float64
	if req.Lat != nil || req.Lon != nil {
		if req.Lat == nil || req.Lon == nil {
			http.Error(w, "lat and lon must be provided together", http.StatusBadRequest)
			return
		}
		lat, lon = req.Lat, req.Lon
	}

	tag, err := h.DB.Exec(ctx, `
		UPDATE businesses SET
			name        = COALESCE($2, name),
			address     = COALESCE($3, address),
			city        = COALESCE($4, city),
			state       = COALESCE($5, state),
			postal_code = COALESCE($6, postal_code),
			is_open     = COALESCE($7, is_open),
			location    = CASE WHEN $8::float8 IS NOT NULL AND $9::float8 IS NOT NULL
			                   THEN ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography
			                   ELSE location END
		WHERE id = $1
	`, id, req.Name, req.Address, req.City, req.State, req.PostalCode, req.IsOpen, lat, lon)

	if err != nil {
		http.Error(w, "failed to update business: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "business not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "updated"})
}

// DeleteBusiness removes the row. The AFTER DELETE trigger writes a
// business_outbox row with operation = 'delete', which the worker uses to
// remove the doc from ES.
func (h *BusinessToESHandler) DeleteBusiness(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tag, err := h.DB.Exec(ctx, `DELETE FROM businesses WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "failed to delete business: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "business not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetOutboxStatus is a debug helper for tests: lets you poll whether the
// outbox row for a business has been processed yet, without querying
// Postgres by hand.
func (h *BusinessToESHandler) GetOutboxStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	rows, err := h.DB.Query(ctx, `
		SELECT id, operation, processed, attempts, created_at
		FROM business_outbox
		WHERE business_id = $1
		ORDER BY id DESC
		LIMIT 10
	`, id)
	if err != nil {
		http.Error(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type outboxStatusRow struct {
		ID        int64  `json:"id"`
		Operation string `json:"operation"`
		Processed bool   `json:"processed"`
		Attempts  int    `json:"attempts"`
		CreatedAt string `json:"created_at"`
	}

	var out []outboxStatusRow
	for rows.Next() {
		var s outboxStatusRow
		var createdAt interface{ String() string }
		_ = createdAt
		var ts pgxTimestamp
		if err := rows.Scan(&s.ID, &s.Operation, &s.Processed, &s.Attempts, &ts); err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.CreatedAt = ts.String()
		out = append(out, s)
	}

	json.NewEncoder(w).Encode(out)
}

// pgxTimestamp is a tiny helper to scan created_at without importing time
// directly into this snippet's signature; swap for time.Time in real code.
type pgxTimestamp struct{ pgx.QueryExecMode }

func (t pgxTimestamp) String() string { return "" } // placeholder — use time.Time in practice
