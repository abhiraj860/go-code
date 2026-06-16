package main

import "fmt"

func binarySearch(nums []int, target int) int {
	s := 0
	e := len(nums) - 1
	for s <= e {
		mid := (s + e) / 2
		if nums[mid] == target {
			return mid
		} else if nums[mid] < target {
			s = mid + 1
		} else {
			e = mid - 1
		}
	}
	return -1
}

func main() {
	nums := []int{-1, 0, 3, 5, 9, 12}
	fmt.Println(binarySearch(nums, 9))
	fmt.Println(binarySearch(nums, -1))
	fmt.Println(binarySearch(nums, 2))
}