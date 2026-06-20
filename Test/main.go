package main

import (
	"fmt"
)

type mapKey struct {
	i int
	j int
}

func main() {
	hashMap := make(map[mapKey]int)
	key := mapKey{0, 1}
	hashMap[key] = 5
	val, found := hashMap[key]
	fmt.Println(val, found)
	fmt.Println(hashMap)
}