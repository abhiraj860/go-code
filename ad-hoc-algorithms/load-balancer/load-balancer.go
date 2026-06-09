package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
)

// -------------------------------------------------------
// 1. Round Robin
// -------------------------------------------------------

type RoundRobin struct {
	servers []string
	counter uint64
}

func NewRoundRobin(servers []string) *RoundRobin {
	return &RoundRobin{servers: servers}
}

// Next atomically increments counter and picks server by modulo.
func (rr *RoundRobin) Next() string {
	i := atomic.AddUint64(&rr.counter, 1)
	return rr.servers[i%uint64(len(rr.servers))]
}

// -------------------------------------------------------
// 2. Weighted Round Robin
// -------------------------------------------------------

type WeightedServer struct {
	addr   string
	weight int
}

type WeightedRoundRobin struct {
	slots   []string // expanded slot list e.g. [A,A,A,B]
	counter uint64
}

func NewWeightedRoundRobin(servers []WeightedServer) *WeightedRoundRobin {
	var slots []string
	for _, s := range servers {
		for i := 0; i < s.weight; i++ {
			slots = append(slots, s.addr)
		}
	}
	return &WeightedRoundRobin{slots: slots}
}

// Next picks the next slot atomically.
func (wrr *WeightedRoundRobin) Next() string {
	i := atomic.AddUint64(&wrr.counter, 1)
	return wrr.slots[i%uint64(len(wrr.slots))]
}

// -------------------------------------------------------
// 3. Least Connections
// -------------------------------------------------------

type LeastConnections struct {
	mu      sync.Mutex
	servers map[string]int // addr -> active connection count
}

func NewLeastConnections(servers []string) *LeastConnections {
	m := make(map[string]int)
	for _, s := range servers {
		m[s] = 0
	}
	return &LeastConnections{servers: m}
}

// Next picks the server with the fewest active connections and increments its count.
func (lc *LeastConnections) Next() string {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	pick, min := "", math.MaxInt
	for addr, count := range lc.servers {
		if count < min {
			min, pick = count, addr
		}
	}
	lc.servers[pick]++
	return pick
}

// Done must be called when a request finishes to decrement the counter.
func (lc *LeastConnections) Done(addr string) {
	lc.mu.Lock()
	lc.servers[addr]--
	lc.mu.Unlock()
}

// -------------------------------------------------------
// 4. IP Hash
// -------------------------------------------------------

type IPHash struct {
	servers []string
}

func NewIPHash(servers []string) *IPHash {
	return &IPHash{servers: servers}
}

// Next hashes the client IP to a fixed server index.
// Same IP always lands on the same server (sticky session).
func (ih *IPHash) Next(clientIP string) string {
	h := fnv.New32a()
	h.Write([]byte(clientIP))
	return ih.servers[h.Sum32()%uint32(len(ih.servers))]
}

// -------------------------------------------------------
// 5. Consistent Hashing
// -------------------------------------------------------

type ConsistentHash struct {
	mu       sync.RWMutex
	ring     map[uint32]string // virtual node hash -> real server
	sorted   []uint32          // sorted virtual node hashes for binary search
	replicas int               // virtual nodes per real server
}

func NewConsistentHash(replicas int) *ConsistentHash {
	return &ConsistentHash{
		ring:     make(map[uint32]string),
		replicas: replicas,
	}
}

func (ch *ConsistentHash) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// Add places a server's virtual nodes onto the ring.
func (ch *ConsistentHash) Add(server string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for i := 0; i < ch.replicas; i++ {
		h := ch.hash(fmt.Sprintf("%s#%d", server, i))
		ch.ring[h] = server
		ch.sorted = append(ch.sorted, h)
	}
	sort.Slice(ch.sorted, func(i, j int) bool { return ch.sorted[i] < ch.sorted[j] })
}

// Remove takes a server's virtual nodes off the ring.
func (ch *ConsistentHash) Remove(server string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for i := 0; i < ch.replicas; i++ {
		h := ch.hash(fmt.Sprintf("%s#%d", server, i))
		delete(ch.ring, h)
	}
	// Rebuild sorted slice without removed hashes
	newSorted := ch.sorted[:0]
	for _, h := range ch.sorted {
		if _, ok := ch.ring[h]; ok {
			newSorted = append(newSorted, h)
		}
	}
	ch.sorted = newSorted
}

// Get finds the first virtual node >= hash(key) on the ring (wraps around).
func (ch *ConsistentHash) Get(key string) string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	h := ch.hash(key)
	i := sort.Search(len(ch.sorted), func(i int) bool { return ch.sorted[i] >= h })
	if i == len(ch.sorted) {
		i = 0 // wrap around the ring
	}
	return ch.ring[ch.sorted[i]]
}

// -------------------------------------------------------
// 6. Power of Two Choices
// -------------------------------------------------------

type PowerOfTwo struct {
	mu      sync.Mutex
	servers map[string]int
	addrs   []string
}

func NewPowerOfTwo(servers []string) *PowerOfTwo {
	m := make(map[string]int)
	for _, s := range servers {
		m[s] = 0
	}
	return &PowerOfTwo{servers: m, addrs: servers}
}

// Next picks 2 random servers and routes to the one with fewer connections.
// O(1) — no full scan, statistically close to Least Connections.
func (pt *PowerOfTwo) Next() string {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	a := pt.addrs[rand.Intn(len(pt.addrs))]
	b := pt.addrs[rand.Intn(len(pt.addrs))]
	pick := a
	if pt.servers[b] < pt.servers[a] {
		pick = b
	}
	pt.servers[pick]++
	return pick
}

// Done decrements the connection count when a request finishes.
func (pt *PowerOfTwo) Done(addr string) {
	pt.mu.Lock()
	pt.servers[addr]--
	pt.mu.Unlock()
}

// -------------------------------------------------------
// Demo
// -------------------------------------------------------

func main() {
	servers := []string{"server-A", "server-B", "server-C"}

	// 1. Round Robin
	rr := NewRoundRobin(servers)
	fmt.Println("=== Round Robin ===")
	for i := 0; i < 6; i++ {
		fmt.Println(rr.Next())
	}

	// 2. Weighted Round Robin
	fmt.Println("\n=== Weighted Round Robin ===")
	wrr := NewWeightedRoundRobin([]WeightedServer{
		{"server-A", 3},
		{"server-B", 1},
		{"server-C", 2},
	})
	for i := 0; i < 6; i++ {
		fmt.Println(wrr.Next())
	}

	// 3. Least Connections
	fmt.Println("\n=== Least Connections ===")
	lc := NewLeastConnections(servers)
	r1 := lc.Next()
	r2 := lc.Next()
	fmt.Println("picked:", r1, r2)
	lc.Done(r1) // r1 finishes
	fmt.Println("after Done(r1), next pick:", lc.Next())

	// 4. IP Hash
	fmt.Println("\n=== IP Hash ===")
	ih := NewIPHash(servers)
	ips := []string{"192.168.1.1", "10.0.0.5", "192.168.1.1"}
	for _, ip := range ips {
		fmt.Printf("IP %s -> %s\n", ip, ih.Next(ip))
	}

	// 5. Consistent Hashing
	fmt.Println("\n=== Consistent Hashing ===")
	ch := NewConsistentHash(100) // 100 virtual nodes per server
	for _, s := range servers {
		ch.Add(s)
	}
	keys := []string{"user:42", "user:99", "order:7"}
	for _, k := range keys {
		fmt.Printf("key %-10s -> %s\n", k, ch.Get(k))
	}
	fmt.Println("removing server-B...")
	ch.Remove("server-B")
	for _, k := range keys {
		fmt.Printf("key %-10s -> %s\n", k, ch.Get(k))
	}

	// 6. Power of Two Choices
	fmt.Println("\n=== Power of Two Choices ===")
	pt := NewPowerOfTwo(servers)
	for i := 0; i < 6; i++ {
		fmt.Println(pt.Next())
	}
}