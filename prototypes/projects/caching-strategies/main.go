package main

import (
	"context"       // carries cancellation info between function calls
	"database/sql"  // talk to Postgres
	"encoding/json" // convert Go structs <-> JSON text
	"errors"        // compare error values
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver; _ = load, never call directly
	"github.com/redis/go-redis/v9"
)

// One product. The backtick parts are "tags" telling the JSON
// encoder what to name each field.
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// Shared handles. The * means "pointer to" — holds the address of
// the real object, not a copy.
var (
	db  *sql.DB
	rdb *redis.Client
)

// How long cached items live before Redis deletes them itself.
const ttl = 60 * time.Second

func main() {
	// An empty context — a required argument carrying "stop now"
	// signals. Background() is the do-nothing version.
	ctx := context.Background()

	var err error
	// sql.Open does NOT connect — it's lazy.
	db, err = sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err) // print and quit
	}
	// defer means "run this when main() ends, whatever happens".
	defer db.Close()

	// Ping forces a real connection so we fail loudly now.
	if err := db.Ping(); err != nil {
		log.Fatal("postgres: ", err)
	}

	// The & means "address of" — build an Options struct, pass its
	// address, which is what the library wants.
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis: ", err)
	}

	setup(ctx)

	fmt.Println("\n=== 1. CACHE-ASIDE (lazy loading) ===")
	demoCacheAside(ctx)

	fmt.Println("\n=== 2. READ-THROUGH ===")
	demoReadThrough(ctx)

	fmt.Println("\n=== 3. WRITE-THROUGH ===")
	demoWriteThrough(ctx)

	fmt.Println("\n=== 4. WRITE-BEHIND (write-back) ===")
	demoWriteBehind(ctx)
}

func setup(ctx context.Context) {
	// IF NOT EXISTS makes this safe to run every time.
	// The backticks make a raw string, so multi-line SQL needs no
	// escaping.
	db.Exec(`CREATE TABLE IF NOT EXISTS products (
		id    SERIAL PRIMARY KEY,
		name  TEXT NOT NULL,
		price NUMERIC(10,2) NOT NULL
	)`)

	// ON CONFLICT DO NOTHING means "skip if the id already exists",
	// so re-running doesn't error.
	db.Exec(`INSERT INTO products (id, name, price) VALUES
		(1, 'keyboard', 49.99),
		(2, 'monitor', 199.00),
		(3, 'mouse', 25.50)
		ON CONFLICT (id) DO NOTHING`)

	// Start each run with an empty cache so the output is predictable.
	rdb.FlushDB(ctx)
	fmt.Println("database seeded, cache cleared")
}

// Build the Redis key name, e.g. "product:1".
func key(id int) string {
	// Sprintf builds a string; %d inserts a number.
	return fmt.Sprintf("product:%d", id)
}

// Read one row straight from Postgres. Every strategy uses this.
func loadFromDB(ctx context.Context, id int) (*Product, error) {
	var p Product
	// $1 is a placeholder Postgres fills with id — this is what
	// makes the query injection-safe.
	// The & before each field is required because Scan WRITES INTO
	// them, so it needs their addresses.
	err := db.QueryRowContext(ctx,
		`SELECT id, name, price FROM products WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Price)
	if err != nil {
		return nil, err
	}
	fmt.Println("    -> hit POSTGRES")
	return &p, nil // & returns p's address
}

// ============ 1. CACHE-ASIDE ============
// The app talks to cache and database separately. The cache has no
// idea Postgres exists. This is the most common strategy.

func cacheAsideGet(ctx context.Context, id int) (*Product, error) {
	// Look in Redis first.
	val, err := rdb.Get(ctx, key(id)).Result()

	// err == nil means found — a cache HIT.
	if err == nil {
		fmt.Println("    -> hit REDIS")
		var p Product
		// Unmarshal parses JSON into the struct; & because it
		// writes into p.
		json.Unmarshal([]byte(val), &p)
		return &p, nil
	}

	// redis.Nil is the special "key doesn't exist" error — a normal
	// miss. errors.Is asks "is this that specific error?", safer
	// than == because errors get wrapped inside others.
	if !errors.Is(err, redis.Nil) {
		// Redis being down shouldn't break reads, so log and fall
		// through to Postgres.
		log.Println("redis error, falling back:", err)
	}

	// MISS: the APPLICATION fetches from the database...
	p, err := loadFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// ...and the APPLICATION populates the cache. That explicit
	// two-step is what makes it "aside".
	data, _ := json.Marshal(p) // _ discards the error
	rdb.Set(ctx, key(id), data, ttl)

	return p, nil
}

func cacheAsideWrite(ctx context.Context, id int, price float64) error {
	// Write to the database first — it's the source of truth.
	_, err := db.ExecContext(ctx,
		`UPDATE products SET price = $1 WHERE id = $2`, price, id)
	if err != nil {
		return err
	}

	// Then DELETE the cached copy, don't update it. Deleting is
	// safer: the next read rebuilds from the database, so you can
	// never cache a wrong value here.
	rdb.Del(ctx, key(id))
	fmt.Println("    -> wrote DB, deleted cache key")
	return nil
}

func demoCacheAside(ctx context.Context) {
	fmt.Println("  read 1 (cold):")
	cacheAsideGet(ctx, 1)

	fmt.Println("  read 2 (warm):")
	cacheAsideGet(ctx, 1)

	fmt.Println("  write:")
	cacheAsideWrite(ctx, 1, 55.00)

	fmt.Println("  read 3 (after write):")
	cacheAsideGet(ctx, 1)
}

// ============ 2. READ-THROUGH ============
// Same behaviour as cache-aside, but the cache logic is hidden
// behind one function. The caller just asks for data and never
// knows a cache exists.

// A "loader" is a function that fetches on a miss. Passing a
// function as a value is what makes this reusable for any type.
type loader func(ctx context.Context, id int) (*Product, error)

// A tiny cache layer. Every caller goes through this — that's the
// difference from cache-aside, where each call site has its own
// get/set code.
type readThroughCache struct {
	load loader
}

func (c *readThroughCache) Get(ctx context.Context, id int) (*Product, error) {
	val, err := rdb.Get(ctx, key(id)).Result()
	if err == nil {
		fmt.Println("    -> hit REDIS")
		var p Product
		json.Unmarshal([]byte(val), &p)
		return &p, nil
	}

	// The cache calls the loader itself. The caller never sees this.
	p, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(p)
	rdb.Set(ctx, key(id), data, ttl)
	return p, nil
}

func demoReadThrough(ctx context.Context) {
	// Wire the loader in once, at construction.
	cache := &readThroughCache{load: loadFromDB}

	fmt.Println("  read 1 (cold):")
	cache.Get(ctx, 2)

	fmt.Println("  read 2 (warm):")
	cache.Get(ctx, 2)

	fmt.Println("  (caller never mentions redis — that's the point)")
}

// ============ 3. WRITE-THROUGH ============
// Every write goes to BOTH the database and the cache, together.
// The cache is never stale, but every write pays the cache cost.

func writeThrough(ctx context.Context, id int, price float64) error {
	// Database first, always.
	_, err := db.ExecContext(ctx,
		`UPDATE products SET price = $1 WHERE id = $2`, price, id)
	if err != nil {
		return err
	}

	// Then read the full row back, so what we cache exactly matches
	// what's stored — including anything the database computed.
	p, err := loadFromDB(ctx, id)
	if err != nil {
		return err
	}

	// UPDATE the cache rather than deleting it. Now the next read is
	// guaranteed to be a hit.
	data, _ := json.Marshal(p)
	rdb.Set(ctx, key(id), data, ttl)

	fmt.Println("    -> wrote DB and cache together")
	return nil
}

func demoWriteThrough(ctx context.Context) {
	fmt.Println("  write:")
	writeThrough(ctx, 3, 30.00)

	fmt.Println("  read (guaranteed hit, no db):")
	cacheAsideGet(ctx, 3)
}

// ============ 4. WRITE-BEHIND (write-back) ============
// Write to the cache and return immediately. The database is
// updated later, in the background. Fastest writes, and the only
// strategy that can LOSE data.

// A pending write waiting to be flushed.
type pendingWrite struct {
	ID    int
	Price float64
}

func writeBehind(ctx context.Context, id int, price float64) error {
	// Update the cache right now, so reads see the new value.
	p, err := loadFromDB(ctx, id)
	if err != nil {
		return err
	}
	p.Price = price
	data, _ := json.Marshal(p)
	rdb.Set(ctx, key(id), data, ttl)

	// Queue the database write for later. A Redis list works as a
	// simple queue; RPush appends to the right end.
	// Using Redis (not an in-memory slice) means a crash doesn't
	// lose the queue — though it's still weaker than writing
	// straight to Postgres.
	job, _ := json.Marshal(pendingWrite{ID: id, Price: price})
	rdb.RPush(ctx, "writequeue", job)

	fmt.Println("    -> wrote cache, queued db write (returned immediately)")
	return nil
}

// The background worker that drains the queue.
func flushWrites(ctx context.Context) {
	for {
		// LPop takes one item from the left end. redis.Nil means
		// the queue is empty.
		job, err := rdb.LPop(ctx, "writequeue").Result()
		if errors.Is(err, redis.Nil) {
			return // nothing left
		}
		if err != nil {
			log.Println("queue error:", err)
			return
		}

		var w pendingWrite
		json.Unmarshal([]byte(job), &w)

		_, err = db.ExecContext(ctx,
			`UPDATE products SET price = $1 WHERE id = $2`, w.Price, w.ID)
		if err != nil {
			// A real implementation would retry or move this to a
			// dead-letter queue. Dropping it loses data silently.
			log.Println("flush failed:", err)
			continue // skip to the next iteration
		}
		fmt.Printf("    -> background: flushed product %d to db\n", w.ID)
	}
}

func demoWriteBehind(ctx context.Context) {
	fmt.Println("  write:")
	writeBehind(ctx, 2, 175.00)

	// Show that the database has NOT caught up yet.
	var dbPrice float64
	db.QueryRowContext(ctx,
		`SELECT price FROM products WHERE id = 2`).Scan(&dbPrice)
	fmt.Printf("  db still says: %.2f  (cache says 175.00)\n", dbPrice)

	fmt.Println("  flushing queue:")
	flushWrites(ctx)

	db.QueryRowContext(ctx,
		`SELECT price FROM products WHERE id = 2`).Scan(&dbPrice)
	fmt.Printf("  db now says: %.2f\n", dbPrice)
}