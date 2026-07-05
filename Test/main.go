package main

import "fmt"

func floor(nums []int, target int) int {
	low := 0
	high := len(nums) - 1
	ans := -1
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] <= target {
			ans = nums[mid]
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return ans
}
func ceil(nums []int, target int) int {
	low := 0
	high := len(nums) - 1
	ans := -1
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] >= target {
			ans = nums[mid]
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return ans
}

func main() {
	nums := []int{2, 4, 6, 8, 10, 12, 14}
	target := 1
	findFloor := floor(nums, target)
	findCeil := ceil(nums, target)

	fmt.Println(findFloor, findCeil)
}