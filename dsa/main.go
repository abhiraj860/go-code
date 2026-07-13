package main

import (
	"fmt"
	"slices"
	"cmp"
)

func smallestString(words []string) string {
	slices.SortFunc(words, func(a,b string) int {
		return cmp.Compare(a + b, b + a)
	})
	res := ""
	for _, w := range words {
		res = res + w
	}
	return res
}

func main() {
	words := []string{"c", "cb", "cba"}
	fmt.Println(smallestString(words))
	words1 := []string{"a", "ab", "aba"}
	fmt.Println(smallestString(words1))
}