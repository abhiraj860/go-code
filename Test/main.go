package main

import (
	"fmt"
)

func main() {
	val := 1
	q := []int{}
	q = append(q, val)
	fmt.Println(q)
	frontVal := q[0]
	fmt.Println(frontVal)
	q = q[1:]
	fmt.Println(q)
}