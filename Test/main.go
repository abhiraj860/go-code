package main 

import (
	"fmt"
	"sync"
)

func fanIn(in chan int, n int) [] chan int {
	outs := make([] chan int, n)
	for i := range n {
		ch := make(chan int)
		outs[i] = ch
		go func(out chan<- int) {
			for v := range in {
				out <- v * v
			}
			close(out)
		}(ch)
	}
	return outs
}

func fanOut(outs [] chan int) chan int {
	var wg sync.WaitGroup
	out := make(chan int)
	for i := range outs {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for v := range outs[i] {
				out <- v
 			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	n := 5
	in := make(chan int, n)
	for i := range n {
		in <- i + 1
	}
	close(in)
	outs := fanIn(in, 3)
	merged := fanOut(outs)

	var result []int
	for p := range merged{
		result = append(result, p)
	}
	fmt.Println(result)
}