package main

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)

type application struct {
	config config
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
