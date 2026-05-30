package main

import "fmt"

func main() {
	nums := []int{2, 3, 4, 5, 5}
	s := 0
	e := len(nums) - 1
	k := 0
	fmt.Scanln(&k)
	found := false
	mid := -1
	for s <= e {
		mid = (s + e) / 2
		if nums[mid] == k {
			found = true
			break
		} else if nums[mid] < k {
			s = mid + 1
		} else {
			e = mid - 1
		}
	}
	if found {
		fmt.Println(mid)
	} else {
		fmt.Println(-1)
	}
}