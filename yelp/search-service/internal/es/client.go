package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const BusinessIndex = "businesses"

// businessIndexMapping defines how the businesses index is structured.
//   - name uses a text field (analyzed, for full-text matching) plus a keyword
//     sub-field (exact match, for sorting/aggregations)
//   - city and categories are keyword fields since we filter on them exactly
//     rather than doing full-text search over them
const businessIndexMapping = `{
  "mappings": {
    "properties": {
      "id":         { "type": "keyword" },
      "name":       {
        "type": "text",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "city":       { "type": "keyword" },
      "state":      { "type": "keyword" },
	  "location":   { "type": "geo_point"},
      "categories": { "type": "keyword" },
      "avg_rating": { "type": "float" },
      "review_count": { "type": "integer" }
    }
  }
}`

// EnsureIndex creates the businesses index with the mapping above,
// if it doesn't already exist. Safe to call on every startup.
func EnsureIndex(ctx context.Context, client *elasticsearch.Client) error {
	existsReq := esapi.IndicesExistsRequest{Index: []string{BusinessIndex}}
	existsRes, err := existsReq.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("checking index existence: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 200 {
		return nil // already exists, nothing to do
	}

	createReq := esapi.IndicesCreateRequest{
		Index: BusinessIndex,
		Body:  bytes.NewReader([]byte(businessIndexMapping)),
	}
	createRes, err := createReq.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("failed to create index: %s", createRes.String())
	}
	return nil
}

// BusinessDoc is what gets indexed into Elasticsearch for each business.
type BusinessDoc struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	CityState   string   `json:"city_state"`
	Location    Location `json:"location"`
	AvgRating   float64  `json:"avg_rating"`
	ReviewCount int      `json:"review_count"`
	Categories  []string `json:"categories"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// IndexBusiness upserts a single business document.
func IndexBusiness(ctx context.Context, client *elasticsearch.Client, doc BusinessDoc) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      BusinessIndex,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(body),
	}
	res, err := req.Do(ctx, client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index business %s: %s", doc.ID, res.String())
	}
	return nil
}
