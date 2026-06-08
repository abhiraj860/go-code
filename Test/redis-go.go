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

type entry struct {
	value string
	expiresAt time.Time
	hasTTL bool
}

type Store struct {
	mu sync.Mutex
	data map[string]entry
}

func NewStore() *Store {
	s := &Store{data:make(map[string]entry)}
	go s.sweep()
	return s
}

func (s *Store) Set(key , val string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{value:val}
	if ttl > 0 {
		e.hasTTL = true
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || (e.hasTTL && time.Now().After(e.expiresAt)) {
		return "", false
	}
	return e.value, true
}

func (s *Store) Del(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	delete(s.data, key)
	return ok
}

func (s * Store) sweep() {
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

func parseCommand(line string, r *bufio.Reader) ([]string, error) {
	if strings.HasPrefix(line, "*") {
		n, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
		args := make([]string, 0, n)
		for i := 0; i < n; i++ {
			r.ReadString('\n')
			val, _ := r.ReadString('\n')
			args = append(args, strings.TrimSpace(val))
		}
		return args, nil
	}
	return strings.Fields(strings.TrimSpace(line)), nil
}

func handle(conn net.Conn, store *Store) {
	defer conn.Close()	
	r := bufio.NewReader(conn)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		args, _ := parseCommand(line, r)
		if len(args) == 0 {
			continue
		}
		cmd := strings.ToUpper(args[0])
		switch cmd {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "SET":
			if len(args) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			var ttl time.Duration
			if len(args) == 5 && strings.ToUpper(args[3]) == "EX" {
				secs, _ := strconv.Atoi(args[4])
				ttl = time.Duration(secs) * time.Second
			}
			store.Set(args[1], args[2], ttl)
			conn.Write([]byte("+OK\r\n"))
		case "GET":
			if len(args) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			if val, ok := store.Get(args[1]); ok {
				fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
			} else {
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
	ln, err := net.Listen("tcp", ":6380")
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on :6380")
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handle(conn, store)
	}
}