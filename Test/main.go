package main

import (
	"fmt"
	"time"
)

func main() {
	go func() {
		fmt.Println("Hello from go routine")
	}()
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Finished main function thread")
}

