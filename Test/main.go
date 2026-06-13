package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	
	go doTask(ctx)
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(2 * time.Second)
}

func doTask(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Task Cancelled")
			return
		default:
			time.Sleep(500 * time.Millisecond)
			fmt.Println("Task Processing...")
		}

	}
}