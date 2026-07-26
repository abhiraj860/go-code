package processor

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/abhiraj/arbiter-lite/internal/metrics"
)

type Ticket struct {
	ID             string  `json:"id"`
	MerchantImpact float64 `json:"merchant_impact"`
	SLARisk        float64 `json:"sla_risk"`
	Sentiment      float64 `json:"sentiment"`
}

type PriorityResult struct {
	TicketID string  `json:"ticket_id"`
	Score    float64 `json:"score"`
	Priority string  `json:"priority"`
}

type Processor struct {
	m *metrics.Metrics
}

func New(m *metrics.Metrics) *Processor {
	return &Processor{m: m}
}

// score stands in for the real Priority Engine — a weighted combination
// of merchant impact, SLA risk, and sentiment.
func score(t Ticket) PriorityResult {
	s := 0.5*t.MerchantImpact + 0.3*t.SLARisk + 0.2*t.Sentiment
	p := "LOW"
	switch {
	case s >= 0.75:
		p = "CRITICAL"
	case s >= 0.5:
		p = "HIGH"
	case s >= 0.25:
		p = "MEDIUM"
	}
	return PriorityResult{TicketID: t.ID, Score: s, Priority: p}
}

func (p *Processor) HandleIngest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		p.m.ObserveDuration(time.Since(start))
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var t Ticket
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		p.m.IncErrors()
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	result := score(t)
	p.m.IncProcessed()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
