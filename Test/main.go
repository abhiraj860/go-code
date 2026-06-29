package main

import (
	"fmt"
)

type lrunode struct {
	key, val int
	prev, next *lrunode
}

type lrucache struct {
	cap int
	cache map[int]*lrunode
	head, tail *lrunode
}

func newlrucache(cap int) *lrucache {
	head := &lrunode{}
	tail := &lrunode{}
	head.next = tail
	tail.prev = head
	return &lrucache{cap:cap, cache: make(map[int]*lrunode), head:head, tail:tail}
}

func (c *lrucache) remove(n * lrunode) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

func (c *lrucache) insertFront(n *lrunode) {
	n.next = c.head.next
	n.prev = c.head
	c.head.next.prev = n
	c.head.next = n
}

func (c *lrucache) moveToFront(n *lrunode) {
	c.remove(n)
	c.insertFront(n)
}

func (c *lrucache) get(key int) int {
	if node, ok := c.cache[key]; ok {
		c.moveToFront(node)
		return node.val
	}
	return -1
}

func (c *lrucache) put(key, val int) {
	if node, ok := c.cache[key]; ok {
		node.val = val
		c.moveToFront(node)
		return 
	}
	if len(c.cache) == c.cap {
		lru := c.tail.prev
		c.remove(lru)
		delete(c.cache, lru.key)
	}
	node := &lrunode{key:key, val:val}
	c.insertFront(node)
	c.cache[key] = node
}


func main() {
	fmt.Println("LRU")
	lru := newlrucache(2)
	lru.put(1, 10)
	lru.put(2, 20)
	fmt.Println(lru.get(1))
	lru.put(3, 30)
	fmt.Println(lru.get(2))
	fmt.Println(lru.get(3))
}















