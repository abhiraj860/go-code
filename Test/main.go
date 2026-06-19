package main

import (
	"fmt"
	"time"
	"sync"
)

func main() {
	i := 0
	j := 0
	k := 0
	var mu sync.Mutex
	go func() {
		for j < 100000 {
			mu.Lock()		
			i += 1
			mu.Unlock()
			j++
		}
	}()
	for k < 100000 {
		mu.Lock()
		i += 1
		mu.Unlock()
		k++
	}
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(i)
}
