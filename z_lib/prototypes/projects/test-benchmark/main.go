package main

import (
	"errors"
	"encoding/json"
	"net/http"
	"strings"
)

var ErrEmpty = errors.New("empty input")

func Normalize(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrEmpty
	}
	return strings.ToLower(s), nil
}

type Todo struct {
	ID int `json:"id"`
	Title string `json:"title"`
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Todo{ID: 1, Title: "ok"})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	http.ListenAndServe(":8080", mux)
}
