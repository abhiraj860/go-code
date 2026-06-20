package main

import (
	"fmt"
	"time"
	"context"
)

func longRunningTask(ctx context.Context) {
	select {
	case <-time.After(20 * time.Millisecond):
		fmt.Println("Task Completed")
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Millisecond)
	defer cancel()
	longRunningTask(ctx)
}

