package main

import (
	"fmt"
	"sync"

)

func process(v int) int {
	return 5 * v
}

func processChannel(ch chan int) []int {
	const conc = 3
	result := make(chan int, conc)
	var wg sync.WaitGroup

	for range conc {
		wg.Add(1)
		go func() {
			wg.Done()
			v := <-ch
			result <- process(v)
		}()
	}
	wg.Wait()
	close(result)
	var out []int
	for v := range result {
		out = append(out, v)
	}
	return out
}

func main() {
	ch := make(chan int, 11)
	for i := range 11 {
		ch <- i + 1
	}
	close(ch)
	result := processChannel(ch)

	for _, v := range result {
		fmt.Println(v)
	}
}
