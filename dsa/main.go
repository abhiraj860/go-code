package main

import (
	"fmt"
)
type pair struct{
	first string
	second int
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func badness(ranks []pair) int {
	 bad := 0
	 desRank := make([]int, len(ranks) + 1)
	 for _, v := range ranks {
		desRank[v.second]++
	 }
	 pos := 0
	 for i := 1; i <= len(ranks); i++ {
		for desRank[i] > 0 {
			pos++
			bad += abs(pos - i)
			desRank[i]--
		}
	 }
	 return bad
}

func main() {
	ranks := []pair{{"A", 1}, {"B", 2}, {"W", 2}, {"B", 1}, {"D", 5}, {"S", 7}, {"W", 7}}
	fmt.Println(badness(ranks))
}