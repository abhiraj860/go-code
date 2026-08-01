package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

type Todo struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Done bool `json:"done"`
}

var (
	db *sql.DB
	rdb *redis.Client
)

const cacheTTL = 60 * time.Second

func main() {
	var err error
	db, err = sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("cannot reach postgres: ", err)
	}
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("cannot reach redis", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /todos/{id}", getTodo)
	mux.HandleFunc("PUT /todos/{id}", updateTodo)
	log.Println("listening on port :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func cacheKey(id string) string {
	return "todo: " + id
}

func getTodo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	key := cacheKey(id)
	cached, err := rdb.Get(ctx, key).Result()
	if err == nil {
		log.Println("cache HIT", key)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}
	if !errors.Is(err, redis.Nil) {
		log.Println("redis error, falling back to db:", err)
	}
	log.Println("cache miss", key)
	var t Todo
	err = db.QueryRowContext(ctx, `SELECT id, title, done FROM todos WHERE id = $1`, id).Scan(&t.ID, &t.Title, &t.Done)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", 404)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	body, jsonErr := json.Marshal(t)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), 500)
		return
	}

	if setErr := rdb.Set(ctx, key, body, cacheTTL).Err(); setErr != nil {
		log.Println("could not cache:", setErr)
	} 
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	res, err := db.ExecContext(ctx, `UPDATE todos SET title = $1, done = $2 WHERE id = $3`, t.Title, t.Done, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", 404)
		return 
	}
	if delErr := rdb.Del(ctx, cacheKey(id)).Err(); delErr != nil {
		log.Println("could not invalidate the cache", delErr)
	} 
	w.WriteHeader(204)
}