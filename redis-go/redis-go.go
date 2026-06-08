package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// entry holds a stored value with an optional TTL.
// hasTTL guards against zero-value time.Time being mistaken for expiry.
type entry struct {
	value     string
	expiresAt time.Time
	hasTTL    bool
}

// Store is the in-memory key-value store.
// RWMutex allows concurrent reads; writes are exclusive.
type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

// NewStore initializes the store and starts the background sweep goroutine.
func NewStore() *Store {
	s := &Store{data: make(map[string]entry)}
	go s.sweep()
	return s
}

// Set writes a key with an optional TTL (0 = no expiry).
func (s *Store) Set(key, val string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{value: val}
	if ttl > 0 {
		e.hasTTL = true
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

// Get returns the value and true if the key exists and hasn't expired.
// Expired keys are treated as missing (lazy expiry).
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || (e.hasTTL && time.Now().After(e.expiresAt)) {
		return "", false
	}
	return e.value, true
}

// Del removes a key and returns whether it existed.
func (s *Store) Del(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	delete(s.data, key) // delete on missing key is a no-op in Go
	return ok
}

// sweep runs every second and proactively removes expired keys.
// This is the "active expiry" path; Get handles the "lazy expiry" path.
func (s *Store) sweep() {
	for range time.Tick(time.Second) {
		s.mu.Lock()
		now := time.Now()
		for k, e := range s.data {
			if e.hasTTL && now.After(e.expiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}

// parseCommand parses a raw line from the client into a string slice of args.
//
// Redis clients (e.g. redis-cli) use the RESP protocol:
//   *3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
//
// Plain telnet / inline commands look like:
//   SET foo bar\r\n
//
// We support both: RESP arrays start with '*', everything else is split on whitespace.
func parseCommand(line string, r *bufio.Reader) ([]string, error) {
	if strings.HasPrefix(line, "*") {
		// RESP array: first line is *<count>, then alternating $<len> / <value> lines
		n, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
		args := make([]string, 0, n)
		for i := 0; i < n; i++ {
			r.ReadString('\n') // discard the $<len> line — we don't validate length here
			val, _ := r.ReadString('\n')
			args = append(args, strings.TrimSpace(val))
		}
		return args, nil
	}
	// Inline: split on whitespace
	return strings.Fields(strings.TrimSpace(line)), nil
}

// handle runs the command loop for a single client connection.
// Each connection gets its own goroutine; the store is shared and thread-safe.
func handle(conn net.Conn, store *Store) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	for {
		// Block until a newline arrives — this is the start of every command
		line, err := r.ReadString('\n')
		if err != nil {
			// Client disconnected or network error — clean exit
			return
		}

		args, _ := parseCommand(line, r)
		if len(args) == 0 {
			continue
		}

		// Commands are case-insensitive per Redis convention
		cmd := strings.ToUpper(args[0])

		switch cmd {

		case "PING":
			// Health check — redis-cli sends PING on connect
			conn.Write([]byte("+PONG\r\n"))

		case "SET":
			// SET key value [EX seconds]
			if len(args) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			var ttl time.Duration
			// Optional EX <seconds> suffix for key expiry
			if len(args) == 5 && strings.ToUpper(args[3]) == "EX" {
				secs, _ := strconv.Atoi(args[4])
				ttl = time.Duration(secs) * time.Second
			}
			store.Set(args[1], args[2], ttl)
			conn.Write([]byte("+OK\r\n"))

		case "GET":
			// GET key -> bulk string or nil bulk ($-1) if missing/expired
			if len(args) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			if val, ok := store.Get(args[1]); ok {
				// RESP bulk string: $<len>\r\n<value>\r\n
				fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
			} else {
				// RESP null bulk string — signals key not found
				conn.Write([]byte("$-1\r\n"))
			}

		case "DEL":
			// DEL key -> :1 if deleted, :0 if key didn't exist
			if len(args) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			if store.Del(args[1]) {
				conn.Write([]byte(":1\r\n")) // RESP integer
			} else {
				conn.Write([]byte(":0\r\n"))
			}

		default:
			fmt.Fprintf(conn, "-ERR unknown command '%s'\r\n", args[0])
		}
	}
}

func main() {
	store := NewStore()

	// Bind on 6380 to avoid conflicting with a real Redis on 6379
	ln, err := net.Listen("tcp", ":6380")
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on :6380")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue // transient accept error — keep the server alive
		}
		// Each client connection is handled concurrently
		go handle(conn, store)
	}
}

// SET foo bar
// +OK

// GET foo
// $3
// bar

// DEL foo
// :1

// GET foo
// $-1

// DEL foo
// :0

// SET foo bar EX 3
// +OK

// GET foo
// $3
// bar

// --- wait 3 seconds ---

// GET foo
// $-1

// SET a 1
// +OK

// SET b 2
// +OK

// GET a
// $1
// 1

// GET b
// $1
// 2

// DEL a
// :1

// GET a
// $-1

// GET b
// $1
// 2

// PING
// +PONG

// GET nonexistent
// $-1

// DEL nonexistent
// :0