package main

import (
	"fmt"
)

func output(nums []int, k int)[][]int {
	result := [][]int{}
	i := 0
	j := 0
	curr := 0
	for j < len(nums) {
		curr += nums[j]
		for curr > k && i < j {
			curr -= nums[i]
			i++
		}
		if curr == k {
			result = append(result, []int{i, j})
		}
		j++
	}
	return result
}

func main() {
	plots := []int{1, 3, 2, 1, 4, 1, 3, 2, 1, 1, 2}
	k := 8
	out := output(plots, k)
	for _, v := range out {
		fmt.Println(v)
	}
}