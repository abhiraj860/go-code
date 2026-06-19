package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	i := 0
	k := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	start := time.Now()
	for range 100000 {
		// wg.Add(1)
		// go func() {
		// 	j := 0
		// 	defer wg.Done()
		// 	local := 0
		// 	for j < 100000 {
		// 		local++	
		// 		j++
		// 	}
		// 	mu.Lock()
		// 	i += local
		// 	mu.Unlock()
		// }()
		j:= 0
		for j < 100000 {
			i++
			j++
		}
	} 
	for k < 100000 {
		mu.Lock()
		i += 1
		mu.Unlock()
		k++
	}
	wg.Wait()
	fmt.Println(time.Since(start))
	fmt.Println(i)
}
