package main

import (
	"log"
	"net/http"
	"os"

	"github.com/abhiraj/arbiter-lite/internal/metrics"
	"github.com/abhiraj/arbiter-lite/internal/processor"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	m := metrics.New()
	p := processor.New(m)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	// Simulated ticket ingestion endpoint — stands in for the Kafka consume loop.
	mux.HandleFunc("/ingest", p.HandleIngest)
	mux.HandleFunc("/metrics", m.Handler())

	log.Printf("arbiter-lite consumer listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
