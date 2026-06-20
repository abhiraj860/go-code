package main

import (
	"fmt"
)

func main() {
	val := 1
	stack := []int{}
	stack = append(stack, val)
	fmt.Println(stack)
	topVal := stack[len(stack) - 1]
	fmt.Println(topVal)
	stack = stack[:len(stack) - 1]
	fmt.Println(stack)
}