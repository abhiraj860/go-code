package main

import (
	"fmt"
)

func partition(nums []int, k int, s int, e int) int {
	i := s - 1
	pivot := nums[e]
	for j := s; j < e; j++{
		if nums[j] < pivot {
			i++
			nums[j], nums[i] = nums[i], nums[j]
		}
	}
	nums[i + 1], nums[e] = nums[e], nums[i + 1]
	return i + 1
}

func quickSelect(nums []int, k int, s int, e int) int {
	p := partition(nums, k, s, e)
	if p == k {
		return nums[p]
	} else if k < p {
		return quickSelect(nums, k, s, p - 1)
	} else {
		return quickSelect(nums, k, p + 1, e)
	}
}

func main() {
	nums := []int{7, 10, 4, 3, 20, 15}
	k := 4
	fmt.Println(quickSelect(nums, k - 1, 0, len(nums) - 1))
}