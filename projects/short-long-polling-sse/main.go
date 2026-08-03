package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	ID int `json:"id"`
	Text string `json:"text"`
	At time.Time `json:"at"`
}

type Broker struct {
	mu sync.Mutex
	messages []Message
	waiters map[chan int]bool
}

var broker = &Broker{
	waiters: make(map[chan int]bool),
}

func (b *Broker) Add(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	m := Message{
		ID: len(b.messages) + 1,
		Text: text,
		At: time.Now(),
	}
	b.messages = append(b.messages, m)
	for ch := range b.waiters {
		select {
		case ch <- len(b.messages):
		default:
		}
	}
}

func (b *Broker) Since(v int) []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	if v < 0 || v >= len(b.messages) {
		return nil
	}
	out := make([]Message, len(b.messages) - v)
	copy(out, b.messages[v:])
	return out
}

func (b *Broker) Version() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

func (b *Broker) Subscribe() chan int {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan int, 1)
	b.waiters[ch] = true
	return ch
}

func (b *Broker) Unsubscribe(ch chan int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.waiters, ch)
}

func parseSince(r *http.Request) int {
	v, _ := strconv.Atoi(r.URL.Query().Get("since"))
	if v < 0 {
		return 0
	}
	return v
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func publish(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" {
		text = "manual message"
	}
	broker.Add(text)
	w.WriteHeader(204)
}


func shortPoll(w http.ResponseWriter, r *http.Request) {
	since := parseSince(r)
	msgs := broker.Since(since)
	log.Printf("short poll: since = %d returned %d messages", since, len(msgs))
	writeJSON(w, map[string]any{
		"version": broker.Version(),
		"messages": msgs,
	})
}


func longPoll(w http.ResponseWriter, r *http.Request) {
	since := parseSince(r)
	if msgs := broker.Since(since); len(msgs) > 0 {
		writeJSON(w, map[string]any{
			"version": broker.Version(),
			"messages": msgs,
		})
		return
	}
	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)
	log.Printf("long poll: waiting, since=%d", since)
	select {
	case <-ch:
		writeJSON(w, map[string]any{
			"version": broker.Version(),
			"messages": broker.Since(since),
		})
	case <-time.After(30 * time.Second):
		log.Println("long poll: timed out")
		writeJSON(w, map[string]any{
			"version": broker.Version(),
			"messages": []Message{},
		})
	case <-r.Context().Done():
		log.Println("long poll: client gone")
	}            
}

func sse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported",500)
		return
	}
	since := parseSince(r)
	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)
	log.Println("sse: client connected")
	for _, m := range broker.Since(since) {
		sendEvent(w, flusher, m)
		since++
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()	
	for {
		select {
		case <-ch:
			for _,m := range broker.Since(since) {
				sendEvent(w, flusher, m)
				since++
			}

		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			log.Println("sse: client disconnected")
			return
		}
	}
}

func sendEvent(w http.ResponseWriter, f http.Flusher, m Message) {
	body, err := json.Marshal(m)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "id: %d\ndata: %s\n\n", m.ID, body)
	f.Flush()
}

func main() {
	go func() {
		i := 0
		for {
			time.Sleep(5 * time.Second)
			i++
			broker.Add(fmt.Sprintf("auto message %d", i))
			log.Printf("published message %d", i)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /short", shortPoll)
	mux.HandleFunc("GET /long", longPoll)
	mux.HandleFunc("GET /events", sse)
	mux.HandleFunc("POST /publish", publish)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
























