package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type SearchHandler struct {
	ES *elasticsearch.Client
}

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type searchResult struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	CityState   string   `json:"city_state"`
	Categories  []string `json:"categories"`
	AvgRating   float64  `json:"avg_rating"`
	ReviewCount int      `json:"review_count"`
	Location    Location `json:"location"`
}

// Search handles GET /search?q=pizza&city=Santa+Barbara&category=Italian
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	location := r.URL.Query().Get("location") // e.g. "los angeles, ca" - matches flexibly against city_state
	category := r.URL.Query().Get("category")

	if strings.TrimSpace(q) == "" {
		http.Error(w, "missing required query param: q", http.StatusBadRequest)
		return
	}

	// build a bool query: must match the keyword, optionally filter by location/category
	must := []map[string]interface{}{
		{
			"match": map[string]interface{}{
				"name": q,
			},
		},
	}
	if location != "" {
		// match (not term) since city_state is a text field - handles case/punctuation differences
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"city_state": location,
			},
		})
	}
	var filter []map[string]interface{}
	if category != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"categories": map[string]interface{}{
					"value":            category,
					"case_insensitive": true,
				},
			},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{

			"bool": map[string]interface{}{
				"must":   must,
				"filter": filter,
			},
		},
		"size": 20,
	}

	body, err := json.Marshal(query)
	if err != nil {
		http.Error(w, "failed to build query", http.StatusInternalServerError)
		return
	}

	req := esapi.SearchRequest{
		Index: []string{"businesses"},
		Body:  strings.NewReader(string(body)),
	}
	res, err := req.Do(context.Background(), h.ES)
	if err != nil {
		http.Error(w, "search request failed", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		respBody, _ := io.ReadAll(res.Body)
		http.Error(w, "elasticsearch error: "+string(respBody), http.StatusInternalServerError)
		return
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source searchResult `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		http.Error(w, "failed to parse es response", http.StatusInternalServerError)
		return
	}

	results := make([]searchResult, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		results = append(results, hit.Source)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
