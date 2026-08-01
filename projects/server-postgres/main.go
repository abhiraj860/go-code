package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Todo struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Done bool `json:"done"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /todos", list)
	mux.HandleFunc("POST /todos", create)
	mux.HandleFunc("PUT /todos/{id}", update)
	mux.HandleFunc("DELETE /todos/{id}", remove)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func list(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(r.Context(), `SELECT id, title, done FROM todos ORDER BY id`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	todos := []Todo{}
	for rows.Next() {
		var t Todo 
		if err := rows.Scan(&t.ID, &t.Title, &t.Done); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		todos = append(todos, t)
	}
	writeJSON(w, 200, todos)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func create(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad json", 400)
		return 
	} 
	err := db.QueryRowContext(r.Context(), `INSERT INTO todos (title, done) VALUES ($1, $2) RETURNING id`, t.Title, t.Done).Scan(&t.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 201, t)
}

func update(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	res, err := db.ExecContext(r.Context(), `UPDATE todos SET title = $1, done = $2 WHERE id = $3`, t.Title, t.Done, r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", 404)
		return 
	}
	w.WriteHeader(204)
}

func remove(w http.ResponseWriter, r *http.Request) {
	res, err := db.ExecContext(r.Context(), `DELETE FROM todos WHERE id = $1`, r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(200)
}

