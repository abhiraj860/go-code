package main

import (
	"fmt"
)

func generate(nums ...int) chan int {
	outs := make(chan int)
	go func() {
		for _, v := range nums {
			outs <- v
		}
		close(outs)
	}() 
	return outs
}

func double(src chan int) chan int {
	out := make(chan int)
	go func() {
		for v := range src {
			out <- 2 * v
		}
		close(out)
	}()
	return out
}

func filter(src chan int, threshold int) chan int {
	out := make(chan int)
	go func() {
		for v := range src {
			if v > threshold {
				out <- v
			}
		}
		close(out)
	}()
	return out
}

func main() {
	src := generate(1, 2, 3, 4, 5)
	transform := double(src)
	filtered := filter(transform, 6)
	result := []int{}
	for v := range filtered {
		result = append(result, v)
	}
	fmt.Println(result)
}