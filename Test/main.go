package main

import (
	"fmt"
	"time"
)

func main() {
	i := 0
	j := 0
	k := 0
	for j < 10000 {
		i += 1
		j++
	}
	for k < 10000 {
		i += 1
		k++
	}
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(i)
}
