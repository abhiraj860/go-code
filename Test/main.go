package main

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"fmt"
)
type HashRing struct {
	ring map[int]string
	sortedPositions []int
	replicationFactor int
}

func NewHashRing(replicationFactor int) *HashRing {
	return &HashRing{
		ring: make(map[int]string),
		replicationFactor: replicationFactor,
	}
}

func hash(key string) int {
	digest := sha256.Sum256([]byte(key))
	return int(binary.BigEndian.Uint64(digest[:8]))
}

func (h *HashRing) AddNode(name string) {
	for i := range h.replicationFactor {
		pos := hash(fmt.Sprintf("%s#%d", name, i))
		h.ring[pos] = name
		h.sortedPositions = append(h.sortedPositions, pos)
	}
	sort.Ints(h.sortedPositions)
}

func (h *HashRing) RemoveNode(name string) {
	for i := range h.replicationFactor {
		pos := hash(fmt.Sprintf("%s#%d", name, i))
		delete(h.ring, pos)
	}
	h.sortedPositions = h.sortedPositions[:0]
	for pos := range h.ring {
		h.sortedPositions = append(h.sortedPositions, pos)
	}
	sort.Ints(h.sortedPositions)
}

func (h *HashRing) GetNode(key string) string {
	if len(h.sortedPositions) == 0 {
		return ""
	}
	pos := hash(key)
	idx := sort.Search(len(h.sortedPositions), func(i int) bool {
		return h.sortedPositions[i] >= pos
	})
	if idx == len(h.sortedPositions) {
		idx = 0
	}
	return h.ring[h.sortedPositions[idx]]
}


func main() {
	ring := NewHashRing(3)
	ring.AddNode("node-A")
	ring.AddNode("node-B")
	ring.AddNode("node-C")

	keys := []string{"user:13", "user:42", "product:42", "order:99", "session:xyz"}
	for _, key := range keys {
		fmt.Printf("%s -> %s\n", key, ring.GetNode(key))
	}
	ring.RemoveNode("node-B")
	for _, key := range keys {
		fmt.Printf("%s -> %s\n", key, ring.GetNode(key))
	}
}