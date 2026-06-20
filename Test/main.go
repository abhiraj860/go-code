package main

import (
	"fmt"
)

func main() {
	set := make(map[int]struct{})
	set[1] = struct{}{}
	set[2] = struct{}{}
	if _, exists := set[1]; exists {
		fmt.Println("1 exists")
	}
}