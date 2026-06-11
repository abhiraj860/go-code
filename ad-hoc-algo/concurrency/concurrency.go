package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// 1. WORKER POOL
//    Use case: thumbnail generation, batch DB writes, rate-limited API calls
//    Pattern: fixed N goroutines pulling from a shared jobs channel.
//    Why not spawn one goroutine per job? Unbounded goroutine creation
//    under high load exhausts memory. Pool caps resource usage.
// ═══════════════════════════════════════════════════════════════

type Job struct {
	ID      int
	Payload string
}

type Result struct {
	JobID  int
	Output string
	Err    error
}

func workerPool() {
	const numWorkers = 3
	const numJobs = 10

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)

	// spawn a fixed number of workers; they live for the duration of the pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// range over channel blocks until a job arrives or the channel is closed;
			// closing jobs is the shutdown signal — no explicit "stop" message needed
			for job := range jobs {
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond) // simulate work
				results <- Result{
					JobID:  job.ID,
					Output: fmt.Sprintf("worker-%d processed job-%d", workerID, job.ID),
				}
			}
		}(w)
	}

	// enqueue all jobs; buffered channel means this doesn't block
	for i := 0; i < numJobs; i++ {
		jobs <- Job{ID: i, Payload: fmt.Sprintf("data-%d", i)}
	}
	close(jobs) // broadcast to all workers: no more jobs coming

	// close results in a separate goroutine so we can range over it below
	// without deadlocking — wg.Wait() would block if we called it here
	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		fmt.Printf("  [worker-pool] %s\n", r.Output)
	}
}

// ═══════════════════════════════════════════════════════════════
// 2. FAN-OUT / FAN-IN  (Pipeline stage)
//    Use case: parallel API enrichment, scatter-gather search
//    Pattern: one input channel fanned out to N goroutines,
//             results merged back into one output channel.
//    Key insight: all N goroutines compete on the same input channel —
//    Go's channel receive is safe for concurrent use; each value is
//    delivered to exactly one receiver (no duplication).
// ═══════════════════════════════════════════════════════════════

// fanOut spawns N goroutines all reading from the same input channel.
// Each goroutine gets a disjoint subset of values (channel guarantees
// exactly-once delivery). Returns N output channels, one per goroutine.
func fanOut(in <-chan int, n int) []<-chan int {
	outs := make([]<-chan int, n)
	for i := 0; i < n; i++ {
		ch := make(chan int)
		outs[i] = ch
		go func(out chan<- int) {
			for v := range in {
				time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
				out <- v * v // simulate enrichment: square the value
			}
			close(out) // signal downstream that this worker is done
		}(ch)
	}
	return outs
}

// fanIn merges N input channels into one output channel.
// One goroutine per input channel forwards values; a coordinator
// goroutine closes the output once all forwarders finish.
func fanIn(ins []<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, in := range ins {
		wg.Add(1)
		go func(ch <-chan int) {
			defer wg.Done()
			for v := range ch {
				out <- v
			}
		}(in)
	}
	// coordinator: close output only after every input is drained
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func fanOutFanIn() {
	in := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		in <- i
	}
	close(in)

	outs := fanOut(in, 3)
	merged := fanIn(outs)

	var results []int
	for v := range merged {
		results = append(results, v)
	}
	fmt.Printf("  [fan-out/fan-in] squared results: %v\n", results)
}

// ═══════════════════════════════════════════════════════════════
// 3. PIPELINE
//    Use case: ETL — read → transform → filter → write
//    Pattern: chain of goroutines connected by channels.
//    Each stage owns one goroutine, reads from upstream channel,
//    writes to downstream channel. Backpressure is natural:
//    a slow stage blocks its upstream via channel send.
// ═══════════════════════════════════════════════════════════════

// generate is the pipeline source — emits each value and closes when done
func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out) // closing propagates the "done" signal downstream
	}()
	return out
}

// double is a transform stage — each value passes through multiplied by 2
func double(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for v := range in {
			out <- v * 2
		}
		close(out)
	}()
	return out
}

// filter is a filter stage — drops values that don't meet the predicate
func filter(in <-chan int, threshold int) <-chan int {
	out := make(chan int)
	go func() {
		for v := range in {
			if v > threshold {
				out <- v
			}
			// filtered-out values are simply not forwarded; no explicit discard needed
		}
		close(out)
	}()
	return out
}

func pipeline() {
	// chain: generate → double → filter(>6)
	// each stage returns a channel; the next stage consumes it
	src := generate(1, 2, 3, 4, 5)
	doubled := double(src)
	filtered := filter(doubled, 6)

	var out []int
	for v := range filtered {
		out = append(out, v)
	}
	fmt.Printf("  [pipeline] after double+filter(>6): %v\n", out)
}

// ═══════════════════════════════════════════════════════════════
// 4. CONTEXT CANCELLATION + TIMEOUT
//    Use case: HTTP handler cancelling downstream DB/RPC calls
//              when client disconnects or request deadline exceeded.
//    Pattern: context.WithTimeout / context.WithCancel creates a
//    deadline that propagates through the call tree via ctx.Done().
//    Every long-running or blocking op should select on ctx.Done().
// ═══════════════════════════════════════════════════════════════

// slowQuery simulates a DB call. It selects on both completion and
// ctx.Done() so it returns immediately if the caller cancels.
func slowQuery(ctx context.Context, id int) (string, error) {
	select {
	case <-time.After(time.Duration(rand.Intn(200)+50) * time.Millisecond):
		return fmt.Sprintf("result-for-%d", id), nil
	case <-ctx.Done():
		// ctx.Err() is either context.DeadlineExceeded or context.Canceled
		return "", ctx.Err()
	}
}

func contextCancellation() {
	// scenario A: generous deadline — query finishes in time
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel() // always defer cancel to release timer resources even on success
	res, err := slowQuery(ctx, 42)
	if err != nil {
		fmt.Printf("  [context] query timed out: %v\n", err)
	} else {
		fmt.Printf("  [context] query succeeded: %s\n", res)
	}

	// scenario B: tight deadline — query will be cancelled mid-flight
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	res2, err2 := slowQuery(ctx2, 99)
	if err2 != nil {
		fmt.Printf("  [context] tight deadline hit: %v\n", err2)
	} else {
		fmt.Printf("  [context] unexpectedly fast: %s\n", res2)
	}
}

// ═══════════════════════════════════════════════════════════════
// 5. SEMAPHORE (bounded concurrency)
//    Use case: limit concurrent outbound HTTP calls, DB connection slots
//    Pattern: buffered channel of size N acts as a semaphore.
//    Sending acquires a slot (blocks when full); receiving releases it.
//    Prefer this over spawning fewer goroutines when each goroutine
//    also needs to do non-blocked work before/after the critical section.
// ═══════════════════════════════════════════════════════════════

func semaphore() {
	const maxConcurrent = 3
	const totalTasks = 9

	// buffered channel of capacity maxConcurrent — the semaphore
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var completed []int

	for i := 0; i < totalTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{}        // acquire: blocks if maxConcurrent goroutines are already inside
			defer func() { <-sem }() // release: runs even if the goroutine panics

			time.Sleep(30 * time.Millisecond) // critical section: simulate HTTP call
			mu.Lock()
			completed = append(completed, id)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("  [semaphore] completed %d tasks with max %d concurrent\n", len(completed), maxConcurrent)
}

// ═══════════════════════════════════════════════════════════════
// 6. ERRGROUP  (structured concurrency with error propagation)
//    Use case: fan-out N subtasks where any single failure should
//              cancel all peers — parallel DB lookups, multi-part uploads.
//    Pattern: manual implementation of golang.org/x/sync/errgroup.
//    A shared context is cancelled on the first error; all goroutines
//    observe ctx.Done() and exit early. g.Wait() returns the first error.
//    This is the idiomatic replacement for manual WaitGroup + error
//    channel collection.
// ═══════════════════════════════════════════════════════════════

// ErrGroup runs goroutines and cancels all peers on the first error.
type ErrGroup struct {
	cancel func()
	wg     sync.WaitGroup
	once   sync.Once
	err    error
}

func NewErrGroup(ctx context.Context) (*ErrGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &ErrGroup{cancel: cancel}, ctx
}

// Go spawns a goroutine. If fn returns an error, it cancels the shared
// context and stores the error (only the first error is kept via sync.Once).
func (g *ErrGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			// once ensures only the first error wins; subsequent errors discarded
			g.once.Do(func() {
				g.err = err
				g.cancel() // cancel shared context — all peers observe ctx.Done()
			})
		}
	}()
}

// Wait blocks until all goroutines finish and returns the first error.
func (g *ErrGroup) Wait() error {
	g.wg.Wait()
	g.cancel() // no-op if already cancelled; releases context resources
	return g.err
}

func fetchUser(ctx context.Context, id int) (string, error) {
	select {
	case <-time.After(time.Duration(rand.Intn(60)+20) * time.Millisecond):
		if id == 3 {
			return "", fmt.Errorf("user %d not found", id) // simulated failure
		}
		return fmt.Sprintf("user-%d", id), nil
	case <-ctx.Done():
		return "", ctx.Err() // peer failed; bail out early
	}
}

func errGroup() {
	g, ctx := NewErrGroup(context.Background())

	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		i := i // capture loop var
		g.Go(func() error {
			user, err := fetchUser(ctx, i)
			if err != nil {
				return err // triggers cancellation for all peers
			}
			results[i] = user
			return nil
		})
	}

	// Wait blocks until all goroutines finish; returns first non-nil error
	if err := g.Wait(); err != nil {
		fmt.Printf("  [errgroup] failed (peer goroutines cancelled): %v\n", err)
	} else {
		fmt.Printf("  [errgroup] all users fetched: %v\n", results)
	}
}

// ═══════════════════════════════════════════════════════════════
// 7. SINGLEFLIGHT  (request coalescing)
//    Use case: thundering herd on cache miss — N concurrent requests
//              for the same key should trigger only one DB fetch,
//              all waiters share the result.
//    Pattern: map of in-flight calls guarded by a mutex.
//    Double-checked lock: check map under lock, if present join the
//    existing call's WaitGroup instead of starting a new fetch.
// ═══════════════════════════════════════════════════════════════

type call struct {
	wg  sync.WaitGroup
	val string
	err error
}

type SingleFlight struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do runs fn for key exactly once while concurrent calls are in-flight.
// Late arrivals find the existing *call in the map, skip fn entirely,
// and block on wg.Wait() until the original caller's fn returns.
func (sf *SingleFlight) Do(key string, fn func() (string, error)) (string, error) {
	sf.mu.Lock()
	if sf.m == nil {
		sf.m = make(map[string]*call)
	}
	if c, ok := sf.m[key]; ok {
		// another goroutine is already fetching this key — join it
		sf.mu.Unlock()
		c.wg.Wait() // block until the in-flight call completes
		return c.val, c.err
	}
	// first caller for this key — register the call, then unlock before
	// executing fn so other goroutines can proceed to the join branch above
	c := &call{}
	c.wg.Add(1)
	sf.m[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn() // only this goroutine runs fn
	c.wg.Done()         // wake all waiters

	// remove from map so the next cache-miss triggers a fresh fetch
	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()

	return c.val, c.err
}

func singleFlight() {
	var sf SingleFlight
	var wg sync.WaitGroup
	var callCount int64 // atomic counter — how many times fn actually ran

	// 10 goroutines all requesting the same key at the same time
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, _ := sf.Do("user:42", func() (string, error) {
				atomic.AddInt64(&callCount, 1)
				time.Sleep(50 * time.Millisecond) // simulate DB fetch
				return "User{id:42, name:Alice}", nil
			})
			_ = val
		}()
	}

	wg.Wait()
	// expect callCount=1: all 10 callers shared one fetch
	fmt.Printf("  [singleflight] 10 callers, actual fetches: %d\n", callCount)
}

// ═══════════════════════════════════════════════════════════════
// 8. RATE LIMITER  (token bucket via ticker)
//    Use case: outbound API calls respecting third-party rate limits.
//    Pattern: time.Ticker fires at a fixed interval; the main loop
//    blocks on ticker.C before dispatching each request goroutine.
//    The loop is sequential on purpose — ticker.C is the gate.
//    Each goroutine is spawned only after a token is available,
//    ensuring at most ratePerSec goroutines start per second.
// ═══════════════════════════════════════════════════════════════

func rateLimiter() {
	const ratePerSec = 5 // allow 5 requests/sec
	const totalReqs = 8

	// ticker fires every (1s / rate) — each tick grants one token
	ticker := time.NewTicker(time.Second / ratePerSec)
	defer ticker.Stop()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < totalReqs; i++ {
		<-ticker.C // block here until the next token; this IS the rate limiter
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("  [rate-limiter] request %d dispatched at %.0fms\n",
				id, float64(time.Since(start).Milliseconds()))
		}(i)
	}

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════
// 9. PUBSUB  (in-process event bus)
//    Use case: decouple producers from consumers — order events,
//              audit logs, metric emissions.
//    Pattern: broker holds per-topic subscriber lists guarded by
//    RWMutex (many concurrent readers, exclusive writes).
//    Each subscriber gets its own buffered channel so Publish is
//    non-blocking as long as the subscriber drains fast enough.
//    CAVEAT: if a slow subscriber fills its buffer, Publish will block.
//    In production, use a select with a default case to drop messages
//    to lagging subscribers instead of blocking the publisher.
// ═══════════════════════════════════════════════════════════════

type Broker struct {
	mu   sync.RWMutex
	subs map[string][]chan string
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[string][]chan string)}
}

// Subscribe returns a dedicated channel for this subscriber.
// Uses a write lock because we're modifying the subscriber list.
func (b *Broker) Subscribe(topic string) <-chan string {
	ch := make(chan string, 10)
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()
	return ch
}

// Publish sends msg to every subscriber's channel.
// Read lock suffices — we're only iterating, not modifying the list.
func (b *Broker) Publish(topic, msg string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[topic] {
		ch <- msg // blocks if this subscriber's buffer is full (see CAVEAT above)
	}
}

// Close signals all subscribers on a topic that no more messages will come.
// Closing the channel causes range loops in subscriber goroutines to exit.
func (b *Broker) Close(topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[topic] {
		close(ch)
	}
	delete(b.subs, topic)
}

func pubSub() {
	broker := NewBroker()

	sub1 := broker.Subscribe("orders")
	sub2 := broker.Subscribe("orders")

	var wg sync.WaitGroup

	// each subscriber runs in its own goroutine, draining its channel
	for idx, sub := range []<-chan string{sub1, sub2} {
		wg.Add(1)
		go func(id int, ch <-chan string) {
			defer wg.Done()
			for msg := range ch { // exits when broker.Close() closes the channel
				fmt.Printf("  [pubsub] subscriber-%d received: %s\n", id, msg)
			}
		}(idx+1, sub)
	}

	broker.Publish("orders", "order-created:101")
	broker.Publish("orders", "order-shipped:101")
	broker.Publish("orders", "order-delivered:101")
	broker.Close("orders") // closes all subscriber channels → range loops exit

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════
// 10. CIRCUIT BREAKER
//     Use case: stop hammering a failing downstream service;
//               fail fast and recover after a cooldown window.
//     States: Closed (normal) → Open (failing fast) → HalfOpen (probing)
//     Transition rules:
//       Closed   → Open     : failures >= threshold
//       Open     → HalfOpen : cooldown elapsed since last failure
//       HalfOpen → Closed   : probe succeeds → reset
//       HalfOpen → Open     : probe fails → reset cooldown timer
// ═══════════════════════════════════════════════════════════════

type State int

const (
	Closed   State = iota // requests pass through normally
	Open                  // requests fail immediately without calling fn
	HalfOpen              // one probe request is allowed through
)

type CircuitBreaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	threshold   int           // number of failures before opening
	lastFailure time.Time
	cooldown    time.Duration // how long to stay Open before probing
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{threshold: threshold, cooldown: cooldown}
}

var ErrCircuitOpen = errors.New("circuit open")

// Call executes fn if the circuit allows it.
// IMPORTANT: we unlock before calling fn to avoid holding the lock
// during potentially slow network calls — holding it would serialise
// all callers and defeat the purpose of the circuit breaker.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	switch cb.state {
	case Open:
		if time.Since(cb.lastFailure) < cb.cooldown {
			cb.mu.Unlock()
			return ErrCircuitOpen // fail fast — don't call fn at all
		}
		// cooldown elapsed: allow one probe through
		cb.state = HalfOpen
	}
	cb.mu.Unlock() // unlock before fn — fn may be slow (network call)

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = Open // threshold hit: open the circuit
		}
		return err
	}
	// success: reset everything — circuit is healthy again
	cb.failures = 0
	cb.state = Closed
	return nil
}

func circuitBreaker() {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	failingService := func() error { return errors.New("service unavailable") }
	healthyService := func() error { return nil }

	// drive 5 calls; circuit opens after 3 failures, remaining calls fast-fail
	for i := 0; i < 5; i++ {
		err := cb.Call(failingService)
		fmt.Printf("  [circuit-breaker] call %d: %v\n", i+1, err)
	}

	// wait for cooldown → circuit transitions to HalfOpen on next call
	time.Sleep(120 * time.Millisecond)
	err := cb.Call(healthyService) // probe succeeds → back to Closed
	fmt.Printf("  [circuit-breaker] after cooldown probe: %v\n", err)
}

// ═══════════════════════════════════════════════════════════════
// 11. SCATTER-GATHER  (parallel fetch with deadline)
//     Use case: search aggregator — fan out to N backends in parallel,
//               collect all that respond within the deadline, drop the rest.
//     Pattern: per-backend goroutine + shared buffered result channel +
//     context timeout. Goroutines that miss the deadline simply return
//     without sending; the coordinator closes the channel after all
//     goroutines finish so the gather loop exits cleanly.
// ═══════════════════════════════════════════════════════════════

type SearchResult struct {
	Source string
	Data   string
}

func searchBackend(ctx context.Context, source string, latency time.Duration) (SearchResult, error) {
	select {
	case <-time.After(latency):
		return SearchResult{Source: source, Data: fmt.Sprintf("results-from-%s", source)}, nil
	case <-ctx.Done():
		return SearchResult{}, fmt.Errorf("%s: %w", source, ctx.Err())
	}
}

func scatterGather() {
	backends := map[string]time.Duration{
		"elasticsearch": 40 * time.Millisecond,
		"postgres":      80 * time.Millisecond,
		"redis":         20 * time.Millisecond,
		"s3":            200 * time.Millisecond, // intentionally slow — will miss deadline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// buffer = len(backends) so fast backends never block on send
	resultCh := make(chan SearchResult, len(backends))
	var wg sync.WaitGroup

	// scatter: all backends start simultaneously
	for source, latency := range backends {
		wg.Add(1)
		go func(src string, lat time.Duration) {
			defer wg.Done()
			res, err := searchBackend(ctx, src, lat)
			if err != nil {
				fmt.Printf("  [scatter-gather] %s missed deadline\n", src)
				return // don't send to resultCh — just exit
			}
			resultCh <- res
		}(source, latency)
	}

	// coordinator: close resultCh once every backend goroutine has returned
	// (either with a result or having missed the deadline)
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// gather: iterate until resultCh is closed
	fmt.Println("  [scatter-gather] results within 100ms deadline:")
	for r := range resultCh {
		fmt.Printf("    ✓ %s → %s\n", r.Source, r.Data)
	}
}

// ═══════════════════════════════════════════════════════════════
// 12. SYNC.ONCE  (lazy singleton initialisation)
//     Use case: initialise a DB connection pool, parse config, load
//               a large model — exactly once regardless of how many
//               goroutines race to use it.
//     Pattern: sync.Once.Do guarantees the function runs exactly once
//     even under concurrent calls. Subsequent calls are no-ops with
//     no locking overhead (once done, the fast path is a single
//     atomic load). Different from a mutex-guarded init: with a mutex
//     you'd re-check the condition on every call; Once eliminates that.
// ═══════════════════════════════════════════════════════════════

type DBPool struct {
	dsn string
}

var (
	dbOnce     sync.Once
	dbInstance *DBPool
)

func getDB() *DBPool {
	dbOnce.Do(func() {
		// expensive initialisation — runs exactly once, even if 100 goroutines call getDB()
		time.Sleep(20 * time.Millisecond) // simulate connection setup
		dbInstance = &DBPool{dsn: "postgres://localhost/prod"}
		fmt.Println("  [sync.once] DB pool initialised")
	})
	return dbInstance // all subsequent calls skip Do entirely — no lock, just atomic load
}

func syncOnce() {
	var wg sync.WaitGroup
	// 10 goroutines race to initialise the DB — only one wins, rest wait then reuse
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db := getDB()
			_ = db
		}()
	}
	wg.Wait()
	fmt.Printf("  [sync.once] 10 goroutines, one init: dsn=%s\n", dbInstance.dsn)
}

// ═══════════════════════════════════════════════════════════════
// 13. ATOMIC OPERATIONS  (lock-free shared state)
//     Use case: hit counters, request metrics, feature flags, sequence
//               number generators — high-contention shared integers.
//     Pattern: sync/atomic operations are hardware-level instructions
//     (LOCK XADD, CMPXCHG) that avoid mutex overhead entirely.
//     Use when: the shared state is a single integer/pointer and you
//     don't need to atomically update multiple fields together.
//     Don't use when: you need to update multiple fields as a unit
//     (use mutex — CAS loops across multiple variables are error-prone).
// ═══════════════════════════════════════════════════════════════

func atomicOps() {
	var (
		requestCount int64 // total requests received
		activeConns  int64 // currently active connections
		featureFlag  int32 // 0=off, 1=on
	)

	var wg sync.WaitGroup

	// simulate 100 concurrent requests: arrive → process → leave
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&requestCount, 1)  // increment — no mutex, no contention
			atomic.AddInt64(&activeConns, 1)   // connection opened
			time.Sleep(time.Millisecond)
			atomic.AddInt64(&activeConns, -1)  // connection closed
		}()
	}

	// CompareAndSwap: atomically set flag 0→1; returns true only for the winner.
	// This is the building block for all lock-free algorithms —
	// "change this value only if it's still what I expect it to be"
	swapped := atomic.CompareAndSwapInt32(&featureFlag, 0, 1)
	fmt.Printf("  [atomic] feature flag CAS succeeded (this goroutine won): %v\n", swapped)

	wg.Wait()

	// Load reads the value atomically — safe even if other goroutines may still be writing
	fmt.Printf("  [atomic] requests=%d  activeConns=%d  flag=%d\n",
		atomic.LoadInt64(&requestCount),
		atomic.LoadInt64(&activeConns),
		atomic.LoadInt32(&featureFlag),
	)
}

// ═══════════════════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════════════════

func section(title string) {
	fmt.Printf("\n┌─ %s\n", title)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	section("1.  Worker Pool          — fixed N goroutines, shared job queue")
	workerPool()

	section("2.  Fan-Out / Fan-In     — parallel processing, merged results")
	fanOutFanIn()

	section("3.  Pipeline             — chained goroutine stages (ETL)")
	pipeline()

	section("4.  Context Cancellation — propagate deadlines and cancellation")
	contextCancellation()

	section("5.  Semaphore            — cap max concurrent goroutines")
	semaphore()

	section("6.  ErrGroup             — structured concurrency, cancel on first error")
	errGroup()

	section("7.  SingleFlight         — coalesce duplicate in-flight requests")
	singleFlight()

	section("8.  Rate Limiter         — token bucket via ticker")
	rateLimiter()

	section("9.  PubSub               — decouple producers from consumers")
	pubSub()

	section("10. Circuit Breaker      — fail fast on degraded downstream")
	circuitBreaker()

	section("11. Scatter-Gather       — parallel fetch, collect within deadline")
	scatterGather()

	section("12. sync.Once            — lazy singleton initialisation")
	syncOnce()

	section("13. Atomic Operations    — lock-free counters and CAS")
	atomicOps()

	section("14. Deadlock              — bank transfer: cause and fix")
	deadlock()
}

// ═══════════════════════════════════════════════════════════════
// 14. DEADLOCK — Bank Transfer (cause and fix)
//
//  SCENARIO: two accounts, two goroutines transferring money to each
//  other simultaneously.
//
//  DEADLOCK CAUSE — lock ordering inconsistency:
//    goroutine A: locks account-1, then tries to lock account-2
//    goroutine B: locks account-2, then tries to lock account-1
//    Both are now waiting for the other to release — forever.
//
//  This is the classic "hold and wait" condition:
//    - goroutine A holds lock-1, waits for lock-2
//    - goroutine B holds lock-2, waits for lock-1
//    Neither can proceed. Go's runtime detects a full deadlock and
//    panics: "all goroutines are asleep — deadlock!"
//
//  THE FIX — consistent global lock ordering:
//    Always lock the account with the lower ID first, regardless of
//    transfer direction. Now both goroutines compete for lock-1 first:
//    one wins, completes the transfer, releases both locks, and the
//    other proceeds. Circular wait is structurally impossible.
//
//  GENERAL RULE: when you must hold multiple locks simultaneously,
//  always acquire them in the same global order everywhere in the codebase.
//  Assign a total order to your locks (e.g. by ID, by pointer address)
//  and never deviate from it.
// ═══════════════════════════════════════════════════════════════

type Account struct {
	id      int
	mu      sync.Mutex
	balance int
}

// transferDeadlock is the BROKEN version.
// It locks `from` first then `to` — order depends on the caller.
// Two concurrent opposite-direction transfers create a circular wait.
func transferDeadlock(from, to *Account, amount int, done chan struct{}) {
	from.mu.Lock()
	// goroutine is now holding from.mu and about to request to.mu.
	// if another goroutine holds to.mu and is waiting for from.mu → deadlock.
	time.Sleep(1 * time.Millisecond) // widen the race window to make deadlock deterministic
	to.mu.Lock()

	from.balance -= amount
	to.balance += amount

	to.mu.Unlock()
	from.mu.Unlock()
	done <- struct{}{}
}

// transferSafe is the FIXED version.
// Always lock the account with the lower ID first, regardless of direction.
// Both goroutines now acquire locks in the same order — circular wait impossible.
func transferSafe(from, to *Account, amount int, done chan struct{}) {
	// enforce global lock order: lower ID always locked first
	first, second := from, to
	if from.id > to.id {
		first, second = to, from // swap so lower-ID account is always locked first
	}

	first.mu.Lock()
	second.mu.Lock()

	from.balance -= amount
	to.balance += amount

	second.mu.Unlock()
	first.mu.Unlock()
	done <- struct{}{}
}

func deadlock() {
	alice := &Account{id: 1, balance: 1000}
	bob := &Account{id: 2, balance: 1000}

	// ── BROKEN: demonstrate the deadlock ──────────────────────────
	fmt.Println("  [deadlock] running BROKEN transfer (will deadlock)...")

	done := make(chan struct{}, 2)
	go transferDeadlock(alice, bob, 100, done) // goroutine A: locks alice → bob
	go transferDeadlock(bob, alice, 200, done) // goroutine B: locks bob  → alice
	// goroutine A holds alice.mu, waiting for bob.mu
	// goroutine B holds bob.mu,   waiting for alice.mu  ← circular wait = deadlock

	// use a timeout to detect the deadlock without hanging the whole demo
	select {
	case <-done:
		<-done
		fmt.Println("  [deadlock] BROKEN: unexpectedly completed (race lost)")
	case <-time.After(50 * time.Millisecond):
		fmt.Println("  [deadlock] BROKEN: confirmed — both goroutines stuck forever")
		fmt.Printf("  [deadlock] BROKEN: alice=%d  bob=%d  (unchanged — no transfer occurred)\n",
			alice.balance, bob.balance)
	}

	// ── FIXED: same scenario with consistent lock ordering ────────
	alice2 := &Account{id: 1, balance: 1000}
	bob2 := &Account{id: 2, balance: 1000}

	fmt.Println("  [deadlock] running FIXED transfer (consistent lock order)...")

	done2 := make(chan struct{}, 2)
	go transferSafe(alice2, bob2, 100, done2) // alice → bob:  internally locks id-1 then id-2
	go transferSafe(bob2, alice2, 200, done2) // bob  → alice: also locks id-1 then id-2
	// both goroutines compete for alice2.mu (id=1) first —
	// one wins, completes fully, releases both; the other then proceeds.
	// No circular wait — structurally impossible with consistent ordering.

	<-done2
	<-done2
	fmt.Printf("  [deadlock] FIXED: both transfers completed successfully\n")
	fmt.Printf("  [deadlock] FIXED: alice=%d  (1000 - 100 + 200 = 1100)\n", alice2.balance)
	fmt.Printf("  [deadlock] FIXED: bob=%d    (1000 - 200 + 100 =  900)\n", bob2.balance)
}