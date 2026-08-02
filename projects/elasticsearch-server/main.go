package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"github.com/elastic/go-elasticsearch/v8"
)

const indexName = "articles"

type Article struct {
	Title string `json:"title"`
	Body string `json:"body"`
}

var es *elasticsearch.Client

func main() {
	var err error
	es, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{os.Getenv("ES_ADDR")},
	})
	if err != nil {
		log.Fatal(err)
	}
	res, err := es.Info()
	if err != nil {
		log.Fatal("cannot reach elasticsearch", err)
	}
	defer res.Body.Close()
	log.Println("Connected to elasticsearch")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /articles", indexArticle)
	mux.HandleFunc("GET /articles/{id}", getArticle)
	mux.HandleFunc("GET /search", search)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func indexArticle(w http.ResponseWriter, r *http.Request) {
	var a Article
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Bad JSON", 400)
		return 
	}
	body, err := json.Marshal(a)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	id := r.URL.Query().Get("id")
	res, err := es.Index(
		indexName,
		bytes.NewReader(body),
		es.Index.WithDocumentID(id),
		es.Index.WithRefresh("true"),
		es.Index.WithContext(r.Context()),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	defer res.Body.Close()
	if res.IsError() {
		http.Error(w, res.String(), 500)
	}
	w.WriteHeader(201)
	w.Write([]byte(`{"status":"indexed"}`))
}

func getArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := es.Get(indexName, id, es.Get.WithContext(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		http.Error(w, "Not Found", 404)
		return
	}
	if res.IsError() {
		http.Error(w, res.String(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	copyBody(w, res.Body)
}

func copyBody(w http.ResponseWriter, body interface{ Read([]byte) (int, error)}) {
	buf := make([]byte, 4096)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			return 
		}
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")	
	query := map[string]any{
		"query": map[string]any{
			"multi_match": map[string]any{
				"query": q,
				"fields": []string{"title^2", "body"},
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	res, err := es.Search(
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
		es.Search.WithContext(r.Context()),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		http.Error(w, res.String(), 500)
		return
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	hits := out["hits"].(map[string]any)["hits"].([]any)
	results := []any{}
	for _, h := range hits {
		doc := h.(map[string]any)
		results = append(results, map[string]any{
			"score": doc["_score"],
			"id": doc["_id"],
			"source": doc["_source"],
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)

}
