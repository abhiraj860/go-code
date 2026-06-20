package main

import (
	"fmt"
)

func main() {
	dq := []int{}
	dq = append(dq, 1, 2, 3)
	from := dq[0]
	back := dq[len(dq) - 1]
	fmt.Println(from, back)
	// popback
	dq = dq[:len(dq) - 1]
	fmt.Println(dq)
	dq = dq[1:]
	fmt.Println(dq)
}