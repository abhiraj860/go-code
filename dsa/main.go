package main

import (
	"fmt"
)

func possible(mid int, arr []int, k int) bool {
	cnt := 1
	curr := 0
	for _, v := range arr {
		if v + curr <= mid {
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

func part(arr []int, k int) int {
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
	arr := []int{5, 10, 30, 20, 15}
	k := 3
	arr2 := []int{10, 20, 30, 40}
	k2 := 2
	arr3 := []int{100, 200, 300, 400}
	k3 := 1
	arr4 := []int{5, 5, 5, 5}
	k4 := 2
	fmt.Println(part(arr, k))
	fmt.Println(part(arr2, k2))
	fmt.Println(part(arr3, k3))
	fmt.Println(part(arr4, k4))

}