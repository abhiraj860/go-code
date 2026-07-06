package main

import (
	"fmt"
)

func midN(mid int, n int, m int) int64 {
	ans := int64(1)
	for i := 1; i <= n; i++ {
		ans = ans * int64(mid)
		if ans > int64(m) {
			return 2
		} 
	}
	if ans == int64(m) {
		return 1
	}
	if ans < int64(m) {
		return 0
	}
	return ans
}

func findNthRoot(n int, m int) int {
	low := 0
	high := m
	for low <= high {
		mid := low + (high - low) / 2
		val := midN(mid, n, m)
		switch val {
		case 1:
			return mid
		case 0:
			low = mid + 1
		default:
			high = mid - 1
		}
	}
	return -1
}

func main() {
	n := 3
	m := 343
	fmt.Println(findNthRoot(n, m))
}