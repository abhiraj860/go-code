package main

import (
	"fmt"
	"time"
)

func main() {
	i := 0
	j := 0
	k := 0
	go func() {
		for j < 100000 {
			i += 1
			j++
		}
	}()
	for k < 100000 {
		i += 1
		k++
	}
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(i)
}
