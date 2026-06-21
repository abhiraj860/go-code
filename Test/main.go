package main

import (
	"fmt"

)

func process(v int) int {
	return 5 * v
}

func processChannel(ch chan int) []int {
	const conc = 10
	result := make(chan int, conc)

	for range conc {
		go func() {
			v := <-ch
			result <- process(v)
		}()
	}
	var out []int
	for range conc {
		out = append(out, <-result)
	}
	return out
}

func main() {
	ch := make(chan int, 10)
	for i := range 10 {
		ch <- i + 1
	}
	result := processChannel(ch)

	for _, v := range result {
		fmt.Println(v)
	}
}
