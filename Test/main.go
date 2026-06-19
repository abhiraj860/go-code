package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Abhiraj")
	go func() {
		fmt.Println("Hello")
	}()		
	time.Sleep(100 * time.Millisecond)

}