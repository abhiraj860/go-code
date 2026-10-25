package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

func main() {
	// Two backend servers, so load balancing is visible.
	go backend(":9001", "A")
	go backend(":9002", "B")
	time.Sleep(200 * time.Millisecond) // let them start

	// Parse both targets.
	a, _ := url.Parse("http://localhost:9001")
	b, _ := url.Parse("http://localhost:9002")
	targets := []*url.URL{a, b}

	// A counter shared across requests. atomic means it's safe when
	// many goroutines touch it at once — each HTTP request runs in
	// its own goroutine.
	var counter uint64

	proxy := &httputil.ReverseProxy{
		// Rewrite runs before each request is forwarded.
		Rewrite: func(r *httputil.ProxyRequest) {
			// Add(&counter, 1) increments and returns the new value.
			// % is remainder, so this cycles 0,1,0,1... — round robin.
			n := atomic.AddUint64(&counter, 1)
			target := targets[n%uint64(len(targets))]

			// SetURL points the outbound request at the backend.
			r.SetURL(target)

			// Tell the backend who the real client was. Without this
			// the backend only sees the proxy's IP.
			r.SetXForwarded()

			// Add our own header, so you can see the proxy touched it.
			r.Out.Header.Set("X-Proxied-By", "demo-proxy")

			log.Printf("%s %s -> %s", r.In.Method, r.In.URL.Path, target.Host)
		},

		// ModifyResponse runs on the way back, before the client
		// sees it. Handy for adding headers or rewriting bodies.
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("X-Served-Via", "proxy")
			return nil
		},

		// If a backend is down, this decides what the client sees.
		// Without it they'd get a bare 502 with no explanation.
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Println("backend failed:", err)
			http.Error(w, "backend unavailable", http.StatusBadGateway)
		},
	}

	log.Println("proxy listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", proxy))
}

// A tiny backend that reports which one it is.
func backend(addr, name string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "handled by backend %s\n", name)
		// Echo back what the proxy added, so you can see it arrive.
		fmt.Fprintf(w, "X-Forwarded-For: %s\n", r.Header.Get("X-Forwarded-For"))
		fmt.Fprintf(w, "X-Proxied-By: %s\n", r.Header.Get("X-Proxied-By"))
	})
	log.Fatal(http.ListenAndServe(addr, mux))
}