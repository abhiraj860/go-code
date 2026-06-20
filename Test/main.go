package main

import (
	"fmt"
	// "time"
)



func main() {
	result := make(chan int)
	go func(a, b int) {
		result <- a + b
	}(3, 5)
	fmt.Println(<-result)
}

