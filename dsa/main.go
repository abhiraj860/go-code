package main

import (
	"fmt"
	"math"
)

func possible(mid int, arr []int, k int) bool {
	cnt := 1
	curr := 0
	for _, v := range arr {
		if curr + v <= mid {
			curr += v
		} else {
			cnt++
			curr = v
		}
	}
	if cnt == k {
		return true
	}
	return false
}

func allocate(arr []int, k int) int {
	if k > len(arr) {
		return -1
	}

	low := math.MinInt
	high := 0
	for _, v := range arr {
		low = max(low, v)
		high += v
	}
	ans := 0
	for low <= high {
		mid := low + (high - low) / 2
		if possible(mid, arr, k) {
			ans = mid
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return ans
}

func main() {
	arr := []int{12, 34, 67, 90}
	k := 2
	fmt.Println(allocate(arr, k))	
}

