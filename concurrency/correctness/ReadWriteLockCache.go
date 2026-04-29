// Go's sync.RWMutex provides RLock()/RUnlock() for read access and Lock()/Unlock() for write access. Multiple goroutines can hold the read lock simultaneously, but the write lock is exclusive.

package correctness

import "sync"

type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func NewCache() *Cache {
    return &Cache{data: make(map[string]string)}
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *Cache) Put(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}
