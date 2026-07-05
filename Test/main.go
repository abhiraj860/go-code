package main

import (
	"fmt"
	"math"
)


func main() {
	nums := []int{3,4,5,6, 1,2}
	low := 0
	high := len(nums) - 1
	ans := math.MaxInt32
	indx := -1
	for low <= high {
		mid := low + (high - low) / 2
		if nums[mid] <= nums[high] {
			if ans >= nums[mid] {
				ans = nums[mid]
				indx = mid
			}
			high = mid - 1
		} else {
			if ans <= nums[low] {
				ans = nums[low]
				indx = low
			}
			low = mid + 1
		}
	}
	fmt.Println(indx)
}