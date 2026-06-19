package main

import (
	"fmt"
	"sync"
	"time"
)

func semaphores(workers int, tasks int) {
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := []int{}
	for t := range tasks {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sem<-struct{}{}
				defer func() {<-sem}()
				mu.Lock()
				time.Sleep(30 * time.Millisecond)				
				result = append(result, t)
				mu.Unlock()
			}(t)	
	}
	wg.Wait()
	fmt.Println(result)
}

func main() {
	totalWorkers := 4
	totalTasks := 12

	semaphores(totalWorkers, totalTasks)
	fmt.Printf("Semaphores with %v workers and %v tasks\n", totalWorkers, totalTasks)
}