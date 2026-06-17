package main

import (
	"fmt"
	"sync"
	"time"
	"math/rand"
)

type Job struct {
	ID int
	Payload string
}

type Result struct {
	JobID int
	Output string
	Err error
}

func workerPool() {
	const numberWorkers = 3
	const numJobs = 10

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)
	
	wg := sync.WaitGroup{}
	for w := range numberWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				results<- Result{
					JobID: job.ID,
					Output: fmt.Sprintf("worker-%d processed job %d", workerID, job.ID),
				}
			}
		}(w)
	}



	for i := range numJobs {
		jobs <- Job{ID: i, Payload: fmt.Sprintf("data-%d", i)}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		fmt.Printf("[worker-pool] %v\n", r.Output)
	}
}

func main() {
	workerPool()
}