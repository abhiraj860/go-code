package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Metrics is a minimal, dependency-free Prometheus-text-format exporter.
// Swap this for github.com/prometheus/client_golang in a real deployment;
// kept stdlib-only here so the project builds with zero external deps.
type Metrics struct {
	processed   int64
	errors      int64
	totalMillis int64
}

func New() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncProcessed() { atomic.AddInt64(&m.processed, 1) }
func (m *Metrics) IncErrors()    { atomic.AddInt64(&m.errors, 1) }
func (m *Metrics) ObserveDuration(d time.Duration) {
	atomic.AddInt64(&m.totalMillis, d.Milliseconds())
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processed := atomic.LoadInt64(&m.processed)
		errs := atomic.LoadInt64(&m.errors)
		total := atomic.LoadInt64(&m.totalMillis)
		var avg float64
		if processed > 0 {
			avg = float64(total) / float64(processed)
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP arbiter_lite_tickets_processed_total Total tickets processed\n")
		fmt.Fprintf(w, "# TYPE arbiter_lite_tickets_processed_total counter\n")
		fmt.Fprintf(w, "arbiter_lite_tickets_processed_total %d\n", processed)
		fmt.Fprintf(w, "# HELP arbiter_lite_process_errors_total Total processing errors\n")
		fmt.Fprintf(w, "# TYPE arbiter_lite_process_errors_total counter\n")
		fmt.Fprintf(w, "arbiter_lite_process_errors_total %d\n", errs)
		fmt.Fprintf(w, "# HELP arbiter_lite_process_duration_ms_avg Average processing duration in ms\n")
		fmt.Fprintf(w, "# TYPE arbiter_lite_process_duration_ms_avg gauge\n")
		fmt.Fprintf(w, "arbiter_lite_process_duration_ms_avg %f\n", avg)
	}
}
