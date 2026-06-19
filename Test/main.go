package main

import (
	"fmt"
	"sync"
)

func main() {
	i := 0
	k := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for cnt := range 6 {
		wg.Add(1)
		go func(id int) {
			j := 0
			defer wg.Done()
			for j < 100000 {
				mu.Lock()		
				i += 1
				mu.Unlock()
				j++
			}
		}(cnt)
	} 
	for k < 100000 {
		mu.Lock()
		i += 1
		mu.Unlock()
		k++
	}
	 wg.Wait()
	fmt.Println(i)
}
