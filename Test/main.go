package main

import (
	"fmt"
)

func twoSum(nums []int, target int) bool {
	i := 0
	j := len(nums) - 1
	for i < j {
		sum := nums[i] + nums[j] 
		if sum == target {
			return true
		} else if sum < target {
			i++
		} else {
			j--
		}
	}
	return false
}

func main() {
	nums1 := []int{1, 3, 4, 6, 8, 10, 13}
	target1 := 13

	nums2 := []int{1, 3, 4, 6, 8, 10, 13}
	target2 := 6
	fmt.Println(twoSum(nums1, target1))
	fmt.Println(twoSum(nums2, target2))
}