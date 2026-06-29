package main

import (
	"fmt"
	"container/list"
)

type LFUEntry struct {
	key, val, freq int
}

type LFUCache struct {
	cap, minFreq int
	keyMap map[int]*list.Element
	freqMap map[int]*list.List
}

func NewLFUCache(cap int) *LFUCache {
	return &LFUCache{
		cap: cap,
		keyMap: make(map[int]*list.Element),
		freqMap: make(map[int]*list.List),
	}
}

func (c *LFUCache) getList(freq int) *list.List {
	if c.freqMap[freq] == nil {
		c.freqMap[freq] = list.New()
	}
	return c.freqMap[freq]
}

func (c *LFUCache) increment(elem *list.Element) {
	entry := elem.Value.(*LFUEntry)
	freq := entry.freq
	c.getList(freq).Remove(elem)
	if c.freqMap[freq].Len() == 0 {
		delete(c.freqMap, freq)
		if c.minFreq == freq {
			c.minFreq++
		}
	}
	entry.freq++
	newElem := c.getList(entry.freq).PushFront(entry)
	c.keyMap[entry.key] = newElem
}

func (c *LFUCache) Get(key int) int {
	elem, ok := c.keyMap[key]
	if !ok {
		return -1
	}
	c.increment(elem)
	return c.keyMap[key].Value.(*LFUEntry).val
}

func (c *LFUCache) Put(key, val int) {
	 if c.cap == 0 {
		return 
	 }
	 if elem, ok := c.keyMap[key]; ok {
		elem.Value.(*LFUEntry).val = val
		c.increment(elem)
		return
	 }
	 if len(c.keyMap) == c.cap {
		lst := c.freqMap[c.minFreq]
		evicted := lst.Back()
		lst.Remove(evicted)
		if lst.Len() == 0 {
			delete(c.freqMap, c.minFreq)
		}
		delete(c.keyMap, evicted.Value.(*LFUEntry).key)
	 }
	 entry := &LFUEntry{key:key, val : val, freq:1}
	 elem := c.getList(1).PushFront(entry)
	 c.keyMap[key] = elem
	 c.minFreq = 1

}


func main() {
	fmt.Println("LFU")
	
}