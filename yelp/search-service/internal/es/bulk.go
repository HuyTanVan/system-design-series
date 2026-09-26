package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// BulkIndexBusinesses indexes many documents in a single request using the
// ES Bulk API. Each doc needs an "action" line (index + id) followed by the
// document body - that's the format ES's _bulk endpoint expects.
func BulkIndexBusinesses(ctx context.Context, client *elasticsearch.Client, docs []BusinessDoc) error {
	var buf bytes.Buffer

	for _, doc := range docs {
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": BusinessIndex,
				"_id":    doc.ID,
			},
		}
		actionLine, err := json.Marshal(action)
		if err != nil {
			return err
		}
		buf.Write(actionLine)
		buf.WriteByte('\n')

		docLine, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body: strings.NewReader(buf.String()),
	}
	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("bulk request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk request returned error status: %s", res.String())
	}

	// Bulk requests return 200 even if individual docs failed - check "errors" field
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int             `json:"status"`
			Error  json.RawMessage `json:"error"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("failed to parse bulk response: %w", err)
	}

	if bulkResp.Errors {
		for _, item := range bulkResp.Items {
			for action, result := range item {
				if result.Status >= 300 {
					return fmt.Errorf(
						"bulk index failed: action=%s status=%d error=%s",
						action,
						result.Status,
						string(result.Error),
					)
				}
			}
		}

		return fmt.Errorf("some documents in the bulk request failed to index")
	}

	return nil
}
