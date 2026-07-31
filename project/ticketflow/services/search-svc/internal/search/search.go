// Package search runs queries against the events index.
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/abhiraj860/ticketflow/services/search-svc/internal/index"
)

// Query describes a search request.
type Query struct {
	// Text is the free-text term. Empty matches everything, which is the
	// browse case.
	Text string
	// City, Kind and Tags are facet filters.
	City string
	Kind int16
	Tags []string
	// From/Size paginate. ElasticSearch's from+size cannot page deeply --
	// beyond ~10k results it degrades badly -- but a storefront never does.
	From int
	Size int
	// OnSaleOnly restricts to events a buyer can act on.
	OnSaleOnly bool
}

// Result is a page of hits plus the facet counts that drive the sidebar.
type Result struct {
	Total  int64
	Events []index.Document
	// Facets are counts per value, keyed by field. The whole reason search
	// lives in ElasticSearch rather than Postgres: computing these in SQL means
	// a GROUP BY per facet per request.
	Facets map[string]map[string]int64
	TookMS int64
}

// Searcher queries the index.
type Searcher struct {
	baseURL string
	http    *http.Client
}

func New(c *index.Client) *Searcher {
	return &Searcher{baseURL: c.BaseURL(), http: c.HTTP()}
}

// Search executes a query and returns hits with facets.
func (s *Searcher) Search(ctx context.Context, q Query) (Result, error) {
	if q.Size <= 0 {
		q.Size = 20
	}
	if q.Size > 100 {
		q.Size = 100
	}
	if q.From < 0 {
		q.From = 0
	}

	body, err := json.Marshal(s.buildQuery(q))
	if err != nil {
		return Result{}, fmt.Errorf("search: marshalling query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/"+index.Name+"/_search", bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("search: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("search: querying: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("search: elasticsearch returned %s: %s", resp.Status, raw)
	}

	var parsed esResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Result{}, fmt.Errorf("search: decoding response: %w", err)
	}

	out := Result{
		Total:  parsed.Hits.Total.Value,
		TookMS: parsed.Took,
		Facets: make(map[string]map[string]int64, len(parsed.Aggregations)),
	}
	for _, hit := range parsed.Hits.Hits {
		out.Events = append(out.Events, hit.Source)
	}
	for name, agg := range parsed.Aggregations {
		buckets := make(map[string]int64, len(agg.Buckets))
		for _, b := range agg.Buckets {
			buckets[string(b.Key)] = b.DocCount
		}
		out.Facets[name] = buckets
	}
	return out, nil
}

// buildQuery assembles the ElasticSearch request body.
func (s *Searcher) buildQuery(q Query) map[string]any {
	// filter rather than must for the facet terms: filters are cacheable by
	// ElasticSearch and contribute no relevance score, which is right because
	// "city = Mumbai" is a yes/no constraint, not a measure of how good a match
	// something is.
	filters := []any{}

	if q.City != "" {
		filters = append(filters, map[string]any{"term": map[string]any{"city": q.City}})
	}
	if q.Kind != 0 {
		filters = append(filters, map[string]any{"term": map[string]any{"kind": q.Kind}})
	}
	for _, tag := range q.Tags {
		filters = append(filters, map[string]any{"term": map[string]any{"tags": tag}})
	}
	if q.OnSaleOnly {
		filters = append(filters, map[string]any{"terms": map[string]any{"status": []int16{1, 2}}})
	}

	var must any
	if q.Text == "" {
		must = map[string]any{"match_all": map[string]any{}}
	} else {
		must = map[string]any{
			"multi_match": map[string]any{
				"query": q.Text,
				// Title is weighted 3x: a user typing "coldplay" wants the
				// Coldplay show, not every event at a venue whose name happens
				// to contain the word.
				"fields": []string{"title^3", "venue_name", "city"},
				// One typo tolerated for short terms, two for longer ones.
				// Users misspell artist names constantly.
				"fuzziness": "AUTO",
			},
		}
	}

	query := map[string]any{
		"from": q.From,
		"size": q.Size,
		"query": map[string]any{
			"bool": map[string]any{"must": must, "filter": filters},
		},
		"aggs": map[string]any{
			"city": map[string]any{"terms": map[string]any{"field": "city", "size": 20}},
			"kind": map[string]any{"terms": map[string]any{"field": "kind", "size": 10}},
			"tags": map[string]any{"terms": map[string]any{"field": "tags", "size": 30}},
		},
	}

	// With no search term, order by soonest -- a browse page is chronological.
	// With a term, let relevance decide.
	if q.Text == "" {
		query["sort"] = []any{map[string]any{"starts_at": "asc"}}
	}

	return query
}

type esResponse struct {
	Took int64 `json:"took"`
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source index.Document `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]struct {
		Buckets []struct {
			Key      facetKey `json:"key"`
			DocCount int64    `json:"doc_count"`
		} `json:"buckets"`
	} `json:"aggregations"`
}

// facetKey decodes an aggregation bucket key.
//
// Needed because ElasticSearch returns the key in the field's own type: a
// quoted string for `city` and `tags`, but a bare number for `kind`, which is
// mapped as a short. A plain string field fails to decode the numeric case,
// and the failure surfaces as an empty facet list rather than an error --
// which looks like "this event has no kind" instead of a parse bug.
type facetKey string

func (k *facetKey) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*k = facetKey(s)
		return nil
	}
	// A bare number, e.g. kind: 1. Keep the literal text.
	*k = facetKey(strings.TrimSpace(string(data)))
	return nil
}
