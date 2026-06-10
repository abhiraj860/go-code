package main

import (
	"container/list"
	"fmt"
	"math/rand"
)

// ─────────────────────────────────────────────
// LRU Cache
// O(1) get/put
// Data structures: doubly linked list + hashmap
// head=MRU end, tail=LRU end (sentinels, never evicted)
// ─────────────────────────────────────────────

type LRUNode struct {
	key, val   int
	prev, next *LRUNode
}

type LRUCache struct {
	cap        int
	cache      map[int]*LRUNode
	head, tail *LRUNode // dummy sentinels; actual nodes sit between them
}

func NewLRUCache(cap int) *LRUCache {
	head := &LRUNode{}
	tail := &LRUNode{}
	head.next = tail
	tail.prev = head
	return &LRUCache{cap: cap, cache: make(map[int]*LRUNode), head: head, tail: tail}
}

// remove unlinks a node from wherever it currently sits in the list
func (c *LRUCache) remove(n *LRUNode) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

// insertFront places a node right after the head sentinel (MRU position)
func (c *LRUCache) insertFront(n *LRUNode) {
	n.next = c.head.next
	n.prev = c.head
	c.head.next.prev = n
	c.head.next = n
}

// moveToFront promotes a node to MRU position on every access
func (c *LRUCache) moveToFront(n *LRUNode) {
	c.remove(n)
	c.insertFront(n)
}

func (c *LRUCache) Get(key int) int {
	if node, ok := c.cache[key]; ok {
		c.moveToFront(node) // access = promote to MRU
		return node.val
	}
	return -1
}

func (c *LRUCache) Put(key, val int) {
	if node, ok := c.cache[key]; ok {
		// key exists: update value and promote
		node.val = val
		c.moveToFront(node)
		return
	}
	if len(c.cache) == c.cap {
		// evict the node just before the tail sentinel (LRU position)
		lru := c.tail.prev
		c.remove(lru)
		delete(c.cache, lru.key)
	}
	node := &LRUNode{key: key, val: val}
	c.insertFront(node)
	c.cache[key] = node
}

// ─────────────────────────────────────────────
// LFU Cache
// O(1) get/put
// Data structures:
//   keyMap:  key -> *list.Element (holds LFUEntry with key/val/freq)
//   freqMap: freq -> doubly linked list of LFUEntries at that freq (front=MRU)
//   minFreq: tracked as a plain int — never needs scanning
//
// minFreq can only:
//   - increment by 1 on Get/Put-update (frequency of accessed key goes up by 1)
//   - reset to 1 on new insert (new keys always start at freq=1)
// This is why we don't need a heap.
// ─────────────────────────────────────────────

type LFUEntry struct {
	key, val, freq int
}

type LFUCache struct {
	cap, minFreq int
	keyMap       map[int]*list.Element // key -> list.Element holding LFUEntry
	freqMap      map[int]*list.List    // freq -> ordered list of LFUEntries (front=MRU)
}

func NewLFUCache(cap int) *LFUCache {
	return &LFUCache{
		cap:     cap,
		keyMap:  make(map[int]*list.Element),
		freqMap: make(map[int]*list.List),
	}
}

// getList lazily initialises the bucket for a given frequency
func (c *LFUCache) getList(freq int) *list.List {
	if c.freqMap[freq] == nil {
		c.freqMap[freq] = list.New()
	}
	return c.freqMap[freq]
}

// increment moves an entry from freqMap[f] to freqMap[f+1]
// and updates minFreq if the old bucket is now empty
func (c *LFUCache) increment(elem *list.Element) {
	entry := elem.Value.(*LFUEntry)
	freq := entry.freq

	// remove from current frequency bucket
	c.getList(freq).Remove(elem)
	if c.freqMap[freq].Len() == 0 {
		delete(c.freqMap, freq)
		// only safe to bump minFreq by 1 because frequency increases by exactly 1
		if c.minFreq == freq {
			c.minFreq++
		}
	}

	// insert at front of next frequency bucket (front = MRU within same freq)
	entry.freq++
	newElem := c.getList(entry.freq).PushFront(entry)
	c.keyMap[entry.key] = newElem // update pointer to new list element
}

func (c *LFUCache) Get(key int) int {
	elem, ok := c.keyMap[key]
	if !ok {
		return -1
	}
	c.increment(elem)
	// read from keyMap[key] not elem — increment replaces the list element
	return c.keyMap[key].Value.(*LFUEntry).val
}

func (c *LFUCache) Put(key, val int) {
	if c.cap == 0 {
		return
	}
	if elem, ok := c.keyMap[key]; ok {
		// key exists: update value and bump frequency
		elem.Value.(*LFUEntry).val = val
		c.increment(elem)
		return
	}
	if len(c.keyMap) == c.cap {
		// evict LRU entry within the lowest-frequency bucket
		// Back() = least recently used among minimum-frequency entries
		lst := c.freqMap[c.minFreq]
		evicted := lst.Back()
		lst.Remove(evicted)
		if lst.Len() == 0 {
			delete(c.freqMap, c.minFreq)
		}
		delete(c.keyMap, evicted.Value.(*LFUEntry).key)
	}
	// new entries always start at freq=1; reset minFreq accordingly
	entry := &LFUEntry{key: key, val: val, freq: 1}
	elem := c.getList(1).PushFront(entry)
	c.keyMap[key] = elem
	c.minFreq = 1
}

// ─────────────────────────────────────────────
// FIFO Cache
// O(1) get/put (amortized for slice dequeue)
// Data structures: hashmap + slice as queue
// Evicts oldest inserted key; access order is irrelevant
// ─────────────────────────────────────────────

type FIFOCache struct {
	cap   int
	cache map[int]int
	queue []int // tracks insertion order; front = oldest
}

func NewFIFOCache(cap int) *FIFOCache {
	return &FIFOCache{cap: cap, cache: make(map[int]int)}
}

func (c *FIFOCache) Get(key int) int {
	if v, ok := c.cache[key]; ok {
		return v // no reordering — FIFO ignores access pattern
	}
	return -1
}

func (c *FIFOCache) Put(key, val int) {
	if _, ok := c.cache[key]; !ok {
		// only enqueue on first insert; updates don't change position
		if len(c.cache) == c.cap {
			evict := c.queue[0] // oldest inserted key
			c.queue = c.queue[1:]
			delete(c.cache, evict)
		}
		c.queue = append(c.queue, key)
	}
	c.cache[key] = val
}

// ────────────────────────────────────────────
// Random Replacement Cache
// O(1) get/put
// Data structures: hashmap + slice of keys (for random index pick)
// Evicts a uniformly random key on capacity overflow
// ─────────────────────────────────────────────

type RRCache struct {
	cap   int
	cache map[int]int
	keys  []int // unordered; used only to pick a random eviction candidate
}

func NewRRCache(cap int) *RRCache {
	return &RRCache{cap: cap, cache: make(map[int]int)}
}

func (c *RRCache) Get(key int) int {
	if v, ok := c.cache[key]; ok {
		return v
	}
	return -1
}

func (c *RRCache) Put(key, val int) {
	if _, ok := c.cache[key]; !ok {
		if len(c.cache) == c.cap {
			idx := rand.Intn(len(c.keys))
			evict := c.keys[idx]
			// swap-delete: move last element into evicted slot, shrink slice
			// O(1) because order in keys slice doesn't matter for RR
			c.keys[idx] = c.keys[len(c.keys)-1]
			c.keys = c.keys[:len(c.keys)-1]
			delete(c.cache, evict)
		}
		c.keys = append(c.keys, key)
	}
	c.cache[key] = val
}

// ─────────────────────────────────────────────
// Main — smoke tests
// ─────────────────────────────────────────────

func main() {
	fmt.Println("=== LRU ===")
	lru := NewLRUCache(2)
	lru.Put(1, 10)
	lru.Put(2, 20)
	fmt.Println(lru.Get(1)) // 10 — promotes key 1 to MRU
	lru.Put(3, 30)          // evicts key 2 (LRU)
	fmt.Println(lru.Get(2)) // -1
	fmt.Println(lru.Get(3)) // 30

	fmt.Println("\n=== LFU ===")
	lfu := NewLFUCache(2)
	lfu.Put(1, 10)
	lfu.Put(2, 20)
	lfu.Get(1)     // freq[1]=2, freq[2]=1
	lfu.Put(3, 30) // evicts key 2 (lowest freq)
	fmt.Println(lfu.Get(2)) // -1
	fmt.Println(lfu.Get(1)) // 10
	fmt.Println(lfu.Get(3)) // 30

	fmt.Println("\n=== FIFO ===")
	fifo := NewFIFOCache(2)
	fifo.Put(1, 10)
	fifo.Put(2, 20)
	fifo.Get(1)     // access doesn't affect eviction order
	fifo.Put(3, 30) // evicts key 1 (oldest inserted)
	fmt.Println(fifo.Get(1)) // -1
	fmt.Println(fifo.Get(2)) // 20
	fmt.Println(fifo.Get(3)) // 30

	fmt.Println("\n=== RR ===")
	rr := NewRRCache(2)
	rr.Put(1, 10)
	rr.Put(2, 20)
	rr.Put(3, 30)          // evicts key 1 or 2 randomly
	fmt.Println(rr.Get(3)) // 30 — always present (just inserted)
}