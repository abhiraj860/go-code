package main

import (
	"fmt"
	"math"
)

func possible(nums []int, k int, mid float64) bool {
	cnt := 0
	for i := 1; i < len(nums); i++ {
		dist := float64(nums[i]) - float64(nums[i - 1])
		numOfStations := int(dist / mid)
		if math.Abs(dist - float64(numOfStations) * mid) < 1e-9 {
			numOfStations--
		}
		cnt += int(numOfStations)
	}

	if cnt > k {
		return false
	}

	return true
}

func dist(nums []int, k int) float64 {
	low := 0.0
	high := float64(math.MinInt)
	for i := 1; i < len(nums); i++ {
		high = max(high, float64(nums[i]) - float64(nums[i - 1]))
	}
	for high - low >= 1e-6 {
		mid := low + (high - low) / 2.0
		if possible(nums, k, mid) {
			high = mid
		} else {
			low = mid
		}
	} 
	return high
}


func main() {
	nums := []int{3, 6, 12, 19, 33}
	k := 3
	nums1 := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	k1 := 1
	fmt.Println(dist(nums, k))
	fmt.Println(dist(nums1, k1))
}