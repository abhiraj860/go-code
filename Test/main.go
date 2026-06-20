package main

import (
	"fmt"
	"sync"
)



func main() {
	var mu sync.Mutex
	counter := 0
	var wg sync.WaitGroup

	for range 1000 {
		wg.Add(1)
		go func() {
			mu.Lock()
			defer wg.Done()
			defer mu.Unlock()
			counter++
		}()
	}
	wg.Wait()
	fmt.Println(counter)
}

