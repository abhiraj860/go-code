package main

import (
	"bytes"         // wrap bytes so they can be read like a small file
	"context"       // carries cancellation and timeout info between calls
	"encoding/json" // convert Go structs <-> JSON text
	"fmt"           // build formatted strings
	"log"           // print messages
	"net/http"      // call Ollama's HTTP API
	"os"            // read environment variables
	"time"          // durations and timeouts

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	collectionName = "documents"

	// nomic-embed-text produces 768 numbers per vector. This MUST
	// match the model exactly — Milvus rejects any other length.
	// Swap models and you must change this and rebuild the collection.
	dim = 768

	// The model Ollama will run.
	embedModel = "nomic-embed-text"
)

// Set once in main so every function can reach them.
var (
	ollamaURL  string
	httpClient *http.Client
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// defer means "run this when main() ends, whatever happens".
	defer cancel()

	// Read addresses from the environment, with local defaults.
	milvusAddr := os.Getenv("MILVUS_ADDR")
	if milvusAddr == "" {
		milvusAddr = "localhost:19530"
	}
	ollamaURL = os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// One shared HTTP client with a timeout. The default client waits
	// FOREVER, which will eventually hang your program.
	// The & means "address of" — we build a struct and pass its address.
	httpClient = &http.Client{Timeout: 60 * time.Second}

	// Check Ollama works BEFORE touching Milvus. If the model isn't
	// pulled, we want that error now, not halfway through inserting.
	if err := checkOllama(ctx); err != nil {
		log.Fatal("ollama: ", err) // print and quit
	}

	// One client, shared. It pools connections internally, so
	// creating one per request would be a real mistake.
	c, err := client.NewClient(ctx, client.Config{Address: milvusAddr})
	if err != nil {
		log.Fatal("connect milvus: ", err)
	}
	defer c.Close()
	log.Println("connected to milvus")

	if err := setupCollection(ctx, c); err != nil {
		log.Fatal("setup: ", err)
	}

	docs := []string{
		"Redis stores data in memory for very fast reads",
		"Postgres is a relational database with ACID transactions",
		"Kafka moves streams of events between services",
		"Kubernetes schedules containers across a cluster of machines",
		"Prometheus collects metrics and fires alerts when they cross thresholds",
		"Nginx sits in front of servers and balances incoming requests",
	}
	if err := insertDocs(ctx, c, docs); err != nil {
		log.Fatal("insert: ", err)
	}

	// These queries share no words with the documents — that's the
	// point. Real embeddings match on meaning, not spelling.
	for _, q := range []string{
		"how do I keep track of how my system is performing",
		"something that handles a lot of traffic quickly",
		"tool for running many containers",
	} {
		if err := search(ctx, c, q, 3); err != nil {
			log.Fatal("search: ", err)
		}
	}
}

// ---------------- OLLAMA ----------------

// The JSON we send. Struct tags in backticks tell the encoder what to
// name each field in the JSON it produces.
type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// The JSON we get back.
type embedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// embed turns text into a vector by asking Ollama's model.
func embed(ctx context.Context, text string) ([]float32, error) {
	// Marshal turns the struct into JSON bytes.
	body, err := json.Marshal(embedRequest{Model: embedModel, Prompt: text})
	if err != nil {
		return nil, err // nil here means "no vector to return"
	}

	// Build the request. bytes.NewReader wraps our bytes so the
	// request can read from them, like a tiny in-memory file.
	req, err := http.NewRequestWithContext(ctx, "POST",
		ollamaURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req) // actually send it
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err) // %w wraps the original
	}
	// Closing the body returns the connection for reuse.
	defer res.Body.Close()

	// A non-200 status means Ollama rejected it — usually a model
	// that hasn't been pulled.
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("ollama returned status %d", res.StatusCode)
	}

	var out embedResponse
	// Decode reads the reply and fills in out. The & is required
	// because Decode WRITES INTO out, so it needs its address.
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	// Catch a dimension mismatch here, with a clear message, instead
	// of getting a confusing rejection from Milvus later.
	if len(out.Embedding) != dim {
		return nil, fmt.Errorf("model returned %d numbers, expected %d — check the dim constant",
			len(out.Embedding), dim)
	}

	// Ollama gives float64; Milvus wants float32. Convert each one.
	// make([]float32, len(...)) creates a list of the right size.
	v := make([]float32, len(out.Embedding))
	for i, x := range out.Embedding { // i is the position, x the value
		v[i] = float32(x) // float32(...) converts the type
	}
	return v, nil
}

// checkOllama proves the model is available before we do real work.
func checkOllama(ctx context.Context) error {
	v, err := embed(ctx, "test")
	if err != nil {
		return err
	}
	log.Printf("ollama ready, model %s returns %d dimensions", embedModel, len(v))
	return nil // nil means "no error"
}

// ---------------- MILVUS SETUP ----------------

func setupCollection(ctx context.Context, c client.Client) error {
	// Check first so restarts are safe.
	has, err := c.HasCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("has collection: %w", err)
	}

	if has {
		log.Println("collection exists — reusing")
		// Loading is needed again after any Milvus restart, and
		// loading an already-loaded collection is harmless.
		return c.LoadCollection(ctx, collectionName, false)
	}

	// The schema is the list of columns and their types.
	// Each .WithX() returns the schema, so calls chain together.
	schema := entity.NewSchema().
		WithName(collectionName).
		WithDescription("documents embedded with ollama").

		// The primary key. AutoID false means we supply the ids.
		WithField(entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeInt64).
			WithIsPrimaryKey(true).
			WithIsAutoID(false)).

		// VarChar needs a maximum length declared up front.
		WithField(entity.NewField().
			WithName("text").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(2048)).

		// The vector column. Dim must match the model's output.
		WithField(entity.NewField().
			WithName("embedding").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(dim))

	// The 2 is the shard count — how many pieces writes are split
	// into. This can't be changed later.
	if err := c.CreateCollection(ctx, schema, 2); err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	log.Println("created collection")

	// An index makes search fast by linking each vector to its near
	// neighbours, so search hops through a graph instead of comparing
	// against everything.
	// entity.COSINE measures angle rather than straight-line distance,
	// which is what text embedding models are trained for.
	// 16 = graph connections per node, 200 = build-time search width.
	idx, err := entity.NewIndexHNSW(entity.COSINE, 16, 200)
	if err != nil {
		return fmt.Errorf("index spec: %w", err)
	}

	// The false means "wait until it's finished".
	if err := c.CreateIndex(ctx, collectionName, "embedding", idx, false); err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	log.Println("created HNSW index")

	// Load pulls the collection into memory. Searching an unloaded
	// collection FAILS — this catches people out constantly.
	if err := c.LoadCollection(ctx, collectionName, false); err != nil {
		return fmt.Errorf("load: %w", err)
	}
	log.Println("collection loaded")

	return nil
}

// ---------------- INSERT ----------------

func insertDocs(ctx context.Context, c client.Client, docs []string) error {
	// Milvus is column-oriented: all ids together, all texts together,
	// all vectors together — NOT one struct per row.
	// make([]T, 0, n) makes an empty list with room already reserved.
	ids := make([]int64, 0, len(docs))
	texts := make([]string, 0, len(docs))
	vecs := make([][]float32, 0, len(docs)) // a list of lists

	for i, d := range docs {
		// Each document needs its own call to the model. This is the
		// slow part — a few hundred milliseconds each on CPU.
		v, err := embed(ctx, d)
		if err != nil {
			return fmt.Errorf("embed %q: %w", d, err)
		}

		ids = append(ids, int64(i+1)) // int64(...) converts the type
		texts = append(texts, d)
		vecs = append(vecs, v)

		log.Printf("embedded %d/%d", i+1, len(docs))
	}

	// Wrap each list in a typed Column so Milvus knows what it holds.
	idCol := entity.NewColumnInt64("id", ids)
	textCol := entity.NewColumnVarChar("text", texts)
	// The dim here must match the schema exactly.
	vecCol := entity.NewColumnFloatVector("embedding", dim, vecs)

	// The "" is the partition name; empty means the default one.
	if _, err := c.Insert(ctx, collectionName, "", idCol, textCol, vecCol); err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	// Flush forces buffered writes to disk so they're searchable now.
	// In production you usually DON'T do this per write — it's
	// expensive, and Milvus flushes on its own schedule.
	if err := c.Flush(ctx, collectionName, false); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	log.Println("inserted", len(docs), "documents")
	return nil
}

// ---------------- SEARCH ----------------

func search(ctx context.Context, c client.Client, query string, topK int) error {
	// The query must go through the SAME model as the documents.
	// A different model puts it in a different vector space and the
	// results become meaningless.
	qv, err := embed(ctx, query)
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}

	// ef controls how widely Milvus explores the graph when
	// searching. Higher is more accurate and slower; it must be at
	// least topK.
	sp, err := entity.NewIndexHNSWSearchParam(64)
	if err != nil {
		return err
	}

	res, err := c.Search(
		ctx,
		collectionName,
		nil,                    // partitions; nil means all of them
		"",                     // filter expression; empty means none
		[]string{"id", "text"}, // which columns to return
		// entity.FloatVector(qv) converts our list into the type
		// Milvus expects. It's a list because you can search several
		// vectors in one call; we send one.
		[]entity.Vector{entity.FloatVector(qv)},
		"embedding",     // which vector column to search
		entity.COSINE,   // must match the index's measure
		topK,            // how many results
		sp,
		// Strong consistency waits for recent writes to be visible.
		// The default is faster but may miss just-inserted rows.
		client.WithSearchQueryConsistencyLevel(entity.ClStrong),
	)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	fmt.Printf("\nquery: %q\n", query)

	// One entry per query vector sent; we sent one.
	for _, r := range res {
		// GetColumn finds a column by name. The .(*entity.ColumnVarChar)
		// is a "type assertion" — the column is a general type, so we
		// state what it really is to read its values.
		// The two-value form with ok avoids a crash if we're wrong.
		textCol, ok := r.Fields.GetColumn("text").(*entity.ColumnVarChar)
		if !ok {
			return fmt.Errorf("unexpected column type")
		}

		for i := 0; i < r.ResultCount; i++ {
			text, err := textCol.ValueByIdx(i)
			if err != nil {
				return err
			}
			// With COSINE, scores run -1 to 1 and HIGHER is more
			// similar — the opposite of L2, where smaller is closer.
			// %.4f prints four decimal places.
			fmt.Printf("  %.4f  %s\n", r.Scores[i], text)
		}
	}
	return nil
}
