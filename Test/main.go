package main

import "fmt"

func lowerBound(nums []int, target int) int {
	low := 0
	high := len(nums) - 1
	ans := len(nums)
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] >= target {
			ans = mid
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return ans
}




func main() {
	nums := []int{2, 3, 7, 10, 11, 11, 25}
	target := 3
	fmt.Println(lowerBound(nums, target))
}