package main

import (
	"database"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func)
}
