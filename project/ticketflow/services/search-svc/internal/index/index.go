// Package index owns the ElasticSearch mapping and the client that reads and
// writes it.
//
// The mapping is the interesting part of a search service. Field types and
// analyzers decide what is findable and what is merely stored, and getting them
// wrong is expensive to correct later: mappings are immutable per field, so a
// change means reindexing the whole corpus.
package index

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Name is the index. Aliased in production so a reindex can swap atomically;
// see EnsureIndex.
const Name = "events"

// Document is one searchable event.
//
// Deliberately denormalised: venue name and city are copied in rather than
// referenced, because a search engine cannot join. The duplication is the
// price of one-round-trip queries, and it is why the indexer must re-read from
// catalog when an event changes.
type Document struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Kind        int16     `json:"kind"`
	Status      int16     `json:"status"`
	VenueID     string    `json:"venue_id"`
	VenueName   string    `json:"venue_name"`
	City        string    `json:"city"`
	CountryCode string    `json:"country_code"`
	StartsAt    time.Time `json:"starts_at"`
	SaleOpensAt time.Time `json:"sale_opens_at"`
	Tags        []string  `json:"tags"`
	PosterURL   string    `json:"poster_url"`
	// MinPriceMinor powers the "from Rs X" badge and price sorting without a
	// second lookup into catalog.
	MinPriceMinor int64  `json:"min_price_minor"`
	CurrencyCode  string `json:"currency_code"`
	Version       int64  `json:"version"`
}

// mapping defines the index.
//
// Analyzer choices, and why:
//
//   - a custom "event_analyzer" with an ASCII-folding filter, so "Beyonce"
//     finds "Beyoncé". Artist names are exactly where diacritics matter and
//     exactly where users will not type them.
//   - title also indexed as a keyword sub-field, because aggregations and exact
//     sorting need an unanalysed value. Analysed text cannot be sorted
//     meaningfully.
//   - city and tags are keyword-only: they are faceted on, never free-text
//     searched, and analysing them would split "New Delhi" into two buckets.
//   - a search-time synonym filter, so "footy" finds football. Search-time
//     rather than index-time means editing the list does not require a
//     reindex.
const mapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "filter": {
        "event_synonyms": {
          "type": "synonym",
          "synonyms": [
            "footy, football, soccer",
            "gig, concert, show",
            "cricket, t20, odi"
          ]
        }
      },
      "analyzer": {
        "event_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        },
        "event_search_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "event_synonyms"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id":            { "type": "keyword" },
      "title": {
        "type": "text",
        "analyzer": "event_analyzer",
        "search_analyzer": "event_search_analyzer",
        "fields": { "keyword": { "type": "keyword", "ignore_above": 256 } }
      },
      "kind":          { "type": "short" },
      "status":        { "type": "short" },
      "venue_id":      { "type": "keyword" },
      "venue_name": {
        "type": "text",
        "analyzer": "event_analyzer",
        "fields": { "keyword": { "type": "keyword", "ignore_above": 256 } }
      },
      "city":            { "type": "keyword" },
      "country_code":    { "type": "keyword" },
      "starts_at":       { "type": "date" },
      "sale_opens_at":   { "type": "date" },
      "tags":            { "type": "keyword" },
      "poster_url":      { "type": "keyword", "index": false },
      "min_price_minor": { "type": "long" },
      "currency_code":   { "type": "keyword" },
      "version":         { "type": "long" }
    }
  }
}`

// Client talks to ElasticSearch over its REST API.
//
// Hand-rolled rather than pulling in the official client: this service issues
// four request shapes, and the official library brings a large dependency tree
// plus its own version-coupling problems for very little here.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("index: base URL is required")
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Ping verifies reachability so a bad address fails at startup.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("index: elasticsearch returned %s", resp.Status)
	}
	return nil
}

// EnsureIndex creates the index when absent. Idempotent, so every replica can
// call it at startup.
func (c *Client) EnsureIndex(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodHead, "/"+Name, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	resp, err = c.do(ctx, http.MethodPut, "/"+Name, strings.NewReader(mapping))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		// Another replica winning the race is not an error.
		if strings.Contains(string(body), "resource_already_exists_exception") {
			return nil
		}
		return fmt.Errorf("index: creating index: %s: %s", resp.Status, body)
	}
	return nil
}

// IndexDocument upserts one document, last write wins.
//
// NO EXTERNAL VERSIONING, deliberately, and this was a bug before it was a
// decision.
//
// The first implementation passed the event's version column as an
// ElasticSearch external version, reasoning that an out-of-order redelivery
// should not overwrite newer data with older. That is sound for a pipeline
// that carries state IN the message. It is wrong here, and harmful:
//
// This indexer is notified by id and then READS CURRENT STATE BACK from
// catalog. Whichever message is processed last therefore holds the freshest
// data by construction, so there is no out-of-order hazard to protect against.
// What versioning did instead was make the index unable to correct itself: a
// document written at version N from a momentarily stale read stayed wrong
// forever, because every later attempt to fix it carried the same version N and
// came back 409 -- which the code then treated as success. Silently.
//
// That failure was observed, not theorised: catalog served a stale cached copy
// once, and the index kept serving a title Postgres had not held for some time,
// with every metric reporting healthy.
//
// Last-write-wins is correct for read-back-on-notify. Convergence comes from
// re-reading the source, not from ordering the writes.
func (c *Client) IndexDocument(ctx context.Context, doc Document) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("index: marshalling document: %w", err)
	}

	path := fmt.Sprintf("/%s/_doc/%s", Name, doc.ID)

	resp, err := c.do(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index: indexing %q: %s: %s", doc.ID, resp.Status, raw)
	}
	return nil
}

// DeleteDocument removes an event, e.g. when it is cancelled.
func (c *Client) DeleteDocument(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/"+Name+"/_doc/"+id, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // already absent; deleting twice is fine
	}
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index: deleting %q: %s: %s", id, resp.Status, raw)
	}
	return nil
}

// Refresh forces pending writes to become searchable.
//
// Only for tests. ElasticSearch is near-real-time by design -- writes become
// visible on the next refresh interval, one second by default -- and calling
// this in production per write destroys indexing throughput.
func (c *Client) Refresh(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodPost, "/"+Name+"/_refresh", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("index: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("index: %s %s: %w", method, path, err)
	}
	return resp, nil
}

// BaseURL exposes the endpoint for the search package.
func (c *Client) BaseURL() string { return c.baseURL }

// HTTP exposes the underlying client for the search package.
func (c *Client) HTTP() *http.Client { return c.http }
