package main

import (
	"fmt"
	"context"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2 *  time.Second)
	defer cancel()
	success := false
	go performTask(ctx, &success)

	
	<-ctx.Done()
	if success == false {
		fmt.Println("Time out limit")
	}

}

func performTask(ctx context.Context, success *bool) {
	<-time.After(5 * time.Second)
	fmt.Println("Task completed successfully")
	*success = true
}