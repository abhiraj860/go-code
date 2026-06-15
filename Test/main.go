package main

import (
	"fmt"
	"math"
)

func maxSum(nums []int, k int) int {
	i, j := 0, 0
	n := len(nums)
	currSum := 0
	maxSum := math.MinInt64
	for j < n {
		currSum += nums[j]
		if j - i + 1 == k {
			maxSum = max(maxSum, currSum)
			currSum -= nums[i]
			i++
		}
		j++
	}
	return maxSum
}

func main() {
	nums := []int{2, 1, 5, 1, 3, 2}
	k := 3
	fmt.Println(maxSum(nums, k))
}