package main

import (
	"fmt"
	"context"
	"sync"
)

func countTo(ctx context.Context, max int, wg *sync.WaitGroup) <-chan int {
	ch := make(chan int)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch)
		for i := range max {
			select {
			case <-ctx.Done():
				fmt.Println("Here")
				return
			case ch<-i:
			}
		}
	}()
	return ch
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	ch := countTo(ctx, 10, &wg)	
	for i := range ch {
		if i > 7 {
			break
		}
		fmt.Println(i)
	}
	cancel()
	wg.Wait()
}
