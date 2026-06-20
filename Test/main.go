package main

import (
	"fmt"
	"sync"
)



func main() {
	var mu sync.RWMutex
	data := make(map[string]int)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		data["key"] = 52
	}()
	wg.Wait()
	mu.RLock()
	value := data["key"]
	mu.RUnlock()
	fmt.Println(value)
}

