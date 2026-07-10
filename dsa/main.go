package main

import (
	"fmt"
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
	if cnt > k {
		return false
	}
	return true
}

func allocate(arr []int, k int) int {
	if k > len(arr) {
		return -1
	}

	low := arr[0]
	high := 0
	for _, v := range arr {
		low = max(low, v)
		high += v
	}
	ans := -1
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
	arr := []int{25, 46, 28, 49, 24}
	k := 4
	fmt.Println(allocate(arr, k))	
}

