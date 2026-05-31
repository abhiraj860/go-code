package main

import "fmt"


func vowelStrings(word string, queries [][]int) []int {
	n := len(word)
	vowelCnt := make([]int, n + 1)
	for i, ch := range word {
		switch ch {
			case 'a', 'e', 'i', 'o', 'u':
				vowelCnt[i + 1] = vowelCnt[i] + 1
			default:
				vowelCnt[i + 1] = vowelCnt[i] 
		}
	}
	res := []int{}
	for _, que := range queries {
		i := que[0]
		j := que[1]
		res = append(res, vowelCnt[j + 1] - vowelCnt[i])
	}
	
	return res
}


func main() {
	word := "prefixsum"
	queries := [][]int{{0, 2}, {1, 4}, {3, 5}}
	fmt.Println(vowelStrings(word, queries))
}