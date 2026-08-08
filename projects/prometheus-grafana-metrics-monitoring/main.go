package main


import (
	"context"   // carries cancellation signals between function calls
	"errors"    // compare error values
	"log"       // print messages
	"math/rand" // fake latency for the demo only
	"net/http"  // build the web server
	"strconv"   // turn numbers into text
	"os/signal" // catch Ctrl+C for a clean shutdown
	"syscall"   // the specific signal names to catch
	"time"      // durations and timing

	"github.com/prometheus/client_golang/prometheus"             // define metrics
	"github.com/prometheus/client_golang/prometheus/collectors"  // built-in Go/process metrics
	"github.com/prometheus/client_golang/prometheus/promhttp"    // serve the /metrics page
)

// Set at build time so a dashboard can show which version is running.
var version = "dev"

// Our own registry rather than the global default. A private registry
// means we control exactly what's exposed — no metrics sneaking in
// from some library we happened to import.
var reg = prometheus.NewRegistry()

// Metrics declared at package level, so they're created once at
// startup rather than per request.
var (
	// A Counter only ever goes UP. Right for "how many requests".
	// Vec means it carries labels — a separate count per combination.
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			// Convention: app_thing_unit_total. The _total suffix is
			// expected on counters.
			Name: "http_requests_total",
			Help: "Total HTTP requests processed.", // shown in the UI
		},
		// Label names. "route" is the PATTERN, never the real URL —
		// see the cardinality warning below.
		[]string{"method", "route", "status"},
	)

	// A Histogram sorts observations into buckets, so you can ask
	// "what's the 95th percentile latency" afterwards.
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "How long HTTP requests took, in seconds.",
			// The bucket edges. Each observation counts in every
			// bucket it's smaller than. Choose these around latencies
			// you actually care about — the defaults assume slow web
			// pages and are wrong for a fast API.
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1, 2.5, 5, 10,
			},
		},
		// Deliberately NO status label. Every label multiplies stored
		// series by the bucket count, and histograms are the
		// expensive metric type.
		[]string{"method", "route"},
	)

	// A Gauge goes up AND down. Right for "how many right now".
	inFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Requests currently being handled.",
		},
	)

	// A business metric rather than a technical one. These are
	// usually what actually matters to whoever is paged at 3am.
	ordersProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_processed_total",
			Help: "Orders processed, by outcome.",
		},
		[]string{"outcome"},
	)

	// The "info" pattern: a gauge always set to 1, whose labels carry
	// the interesting values. Lets dashboards join on version.
	buildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_build_info",
			Help: "Build information; always 1.",
		},
		[]string{"version"},
	)
)

// init runs automatically before main. Registering here means a
// duplicate metric name crashes at startup, not mid-shift.
func init() {
	// MustRegister panics on failure. A "Must" prefix means "this
	// can't sensibly fail, so die if it does".
	reg.MustRegister(
		httpRequests, httpDuration, inFlight, ordersProcessed, buildInfo,

		// Go runtime internals: goroutines, memory, GC pauses.
		// Free, and the first thing you want during an incident.
		collectors.NewGoCollector(),

		// Process-level: CPU, resident memory, open file descriptors.
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// WithLabelValues picks the specific labelled metric.
	// Set puts a gauge at an exact value.
	buildInfo.WithLabelValues(version).Set(1)
}

func main() {
	// A context that cancels on Ctrl+C, so we shut down without
	// cutting off in-flight requests.
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	// defer means "run this when main() ends, whatever happens".
	defer stop()

	// ---- Application server, public port ----
	appMux := http.NewServeMux()

	// Each route is wrapped in instrument(), which times and counts
	// it. The route NAME is passed explicitly, not read from the URL.
	appMux.HandleFunc("GET /api/orders", instrument("/api/orders", handleOrders))
	appMux.HandleFunc("POST /api/orders", instrument("/api/orders", handleCreateOrder))
	appMux.HandleFunc("GET /api/slow", instrument("/api/slow", handleSlow))
	appMux.HandleFunc("GET /healthz", handleHealth) // deliberately NOT instrumented

	// ---- Metrics on a SEPARATE admin port ----
	// Keeping /metrics off the public port means you can't
	// accidentally expose internals, and a flood of user traffic
	// can't stop Prometheus from scraping you.
	adminMux := http.NewServeMux()
	adminMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		// If one metric fails to gather, serve the rest rather than
		// returning nothing.
		ErrorHandling: promhttp.ContinueOnError,
	}))
	adminMux.HandleFunc("/healthz", handleHealth)

	appSrv := newServer(":8080", appMux)
	adminSrv := newServer(":9100", adminMux)

	// go starts a function in the background. A goroutine is a
	// lightweight independent task Go manages for you.
	go func() {
		log.Println("app listening on :8080")
		// ErrServerClosed is the normal result of a graceful stop, so
		// it isn't a real failure. errors.Is asks "is this that
		// specific error?" — safer than == because errors get wrapped.
		if err := appSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("app server: ", err)
		}
	}()

	go func() {
		log.Println("metrics on :9100/metrics")
		if err := adminSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("admin server: ", err)
		}
	}()

	// Fake background work so the graphs have data without you
	// curling constantly.
	go simulateOrders(ctx)

	// Block until Ctrl+C. Done() returns a channel that closes on
	// cancellation; <- waits to receive from it.
	<-ctx.Done()
	log.Println("shutting down")

	// Give in-flight requests 10 seconds to finish before force-quit.
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	appSrv.Shutdown(shutCtx)
	adminSrv.Shutdown(shutCtx)
	log.Println("stopped")
}

// newServer builds a server with timeouts. The default
// http.ListenAndServe has NONE, so one slow client can hold a
// connection open forever.
func newServer(addr string, h http.Handler) *http.Server {
	// The & means "address of" — we build a struct and return its
	// address, which is what the * in the return type expects.
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// ---------------- INSTRUMENTATION ----------------

// A wrapper that remembers the status code. The plain ResponseWriter
// doesn't expose it, so we intercept the call.
type statusWriter struct {
	// Embedding: we get every ResponseWriter method for free and
	// only override the one we care about.
	http.ResponseWriter
	status int
}

// WriteHeader is called when the handler sets a status code.
// The (w *statusWriter) part is the receiver — the value the method
// runs on. The * means changes stick to the real object.
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code) // pass it along
}

// Write catches handlers that write a body without ever calling
// WriteHeader, which implicitly means 200.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.ResponseWriter.Write(b)
}

// instrument wraps a handler with timing and counting.
// It takes a function and returns a function — that's middleware.
func instrument(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Inc adds 1; Dec subtracts 1. The deferred Dec runs even if
		// the handler panics, so the gauge can't drift upward forever.
		inFlight.Inc()
		defer inFlight.Dec()

		sw := &statusWriter{ResponseWriter: w}

		next(sw, r) // run the real handler

		if sw.status == 0 {
			sw.status = 200 // handler wrote nothing at all
		}

		// Since gives the time elapsed since start.
		// .Seconds() converts it to a plain number.
		elapsed := time.Since(start).Seconds()

		// CRITICAL: `route` is the pattern we passed in, NOT r.URL.Path.
		// Using the real path would create one metric series per
		// unique URL — /api/orders/1, /api/orders/2, forever. That's
		// a cardinality explosion, and it's the number one way people
		// take down their own Prometheus.
		httpRequests.WithLabelValues(r.Method, route,
			strconv.Itoa(sw.status)).Inc() // Itoa turns an int into text

		// Observe records one measurement into the histogram.
		httpDuration.WithLabelValues(r.Method, route).Observe(elapsed)
	}
}

// ---------------- HANDLERS ----------------

func handleOrders(w http.ResponseWriter, r *http.Request) {
	// Pretend to do work.
	time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
	w.Write([]byte(`{"orders":[]}`)) // []byte converts text to raw bytes
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.Intn(80)) * time.Millisecond)

	// Fail roughly 15% of the time, so error-rate graphs have
	// something to show.
	if rand.Float64() < 0.15 {
		ordersProcessed.WithLabelValues("failed").Inc()
		http.Error(w, "payment declined", 402)
		return
	}

	ordersProcessed.WithLabelValues("success").Inc()
	w.WriteHeader(201) // 201 = created
	w.Write([]byte(`{"status":"created"}`))
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	// Deliberately slow, so the p95 latency graph has a visible tail.
	time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
	w.Write([]byte(`{"status":"finally done"}`))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Health checks fire constantly and would drown out real traffic
	// in the metrics, which is why this route isn't instrumented.
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

// ---------------- BACKGROUND TRAFFIC ----------------

// simulateOrders generates activity so the dashboards aren't empty.
func simulateOrders(ctx context.Context) {
	// A Ticker fires on a channel at a fixed interval.
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop() // release it when we return

	for {
		// select waits on several channels and takes whichever fires
		// first, so shutdown isn't delayed by the tick interval.
		select {
		case <-ctx.Done(): // fires on Ctrl+C
			return
		case <-t.C: // fires every 300ms
			if rand.Float64() < 0.1 {
				ordersProcessed.WithLabelValues("failed").Inc()
			} else {
				ordersProcessed.WithLabelValues("success").Inc()
			}
		}
	}
}
