package main

import (
	"fmt"
	"time"
)



func main() {
	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(20 * time.Millisecond)
		ch1 <- "hello"
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		ch2 <- "worls"
	}()

	select {
	case msg1 := <-ch1:
		fmt.Println("Msg1 ", msg1)
	case msg1 := <- ch2:
		fmt.Println("Msg2", msg1)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("Time Out")
	}
}

