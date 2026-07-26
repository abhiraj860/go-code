package processor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhiraj/arbiter-lite/internal/metrics"
)

func TestScore(t *testing.T) {
	cases := []struct {
		name string
		in   Ticket
		want string
	}{
		{"critical", Ticket{ID: "1", MerchantImpact: 1, SLARisk: 1, Sentiment: 1}, "CRITICAL"},
		{"low", Ticket{ID: "2", MerchantImpact: 0, SLARisk: 0, Sentiment: 0}, "LOW"},
		{"medium", Ticket{ID: "3", MerchantImpact: 0.3, SLARisk: 0.3, Sentiment: 0.3}, "MEDIUM"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := score(c.in)
			if got.Priority != c.want {
				t.Errorf("score(%v) priority = %s, want %s", c.in, got.Priority, c.want)
			}
		})
	}
}

func TestHandleIngest(t *testing.T) {
	p := New(metrics.New())
	body, _ := json.Marshal(Ticket{ID: "abc", MerchantImpact: 0.9, SLARisk: 0.8, Sentiment: 0.7})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	p.HandleIngest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result PriorityResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.TicketID != "abc" {
		t.Errorf("expected ticket_id abc, got %s", result.TicketID)
	}
}

func TestHandleIngest_BadPayload(t *testing.T) {
	p := New(metrics.New())
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	p.HandleIngest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
