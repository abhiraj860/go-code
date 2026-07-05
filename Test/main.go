package main

import "fmt"

func findLow(nums []int, target int) int {
	low := 0
	high := len(nums) - 1
	ans := -1
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] == target {
			ans = mid
			high = mid - 1
		} else if nums[mid] > target {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return ans
}

func findHigh(nums []int, target int) int {
	low := 0
	high := len(nums) - 1
	ans := -1
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] == target {
			ans = mid
			low = mid + 1
		} else if nums[mid] < target {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return ans
}


func main() {
	nums := []int{8, 9, 10, 12, 12, 12}
	target := 12
	low := findLow(nums, target)
	high := findHigh(nums, target)
	if low == -1 {
		fmt.Println(0)
	} else {
		fmt.Println(high - low + 1)
	}
}