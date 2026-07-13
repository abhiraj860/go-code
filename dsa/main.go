package main

import (
	"fmt"
)

func findShort(nums []int) int {
	s := 0
	n := len(nums)
	for s < n  {
		if s < n - 1 && nums[s] > nums[s + 1] {
			break	
		}
		s++ 	
	}
	if s == n {
		return 0
	}
	e := len(nums) - 1
	for e > 0 {
		if nums[e] < nums[e - 1] {
			break
		} 
		e--
	}
	maxm := nums[s]
	minm := nums[s]
	for k := s + 1; k <= e; k++ {
		maxm = max(maxm, nums[k])
		minm = min(minm, nums[k])
	}
	i := 0
	for i <= s {
		if nums[i] > minm {
			break
		}
		i++
	}
	j := n - 1
	for j >= e {
		if nums[j] < maxm {
			break
		}
		j--
	}
	return j - i + 1

}

func main() {
	nums := []int{2, 6, 4, 8, 10, 9, 15}
	fmt.Println(findShort(nums))
	nums2 := []int{1, 2, 3, 4}
	fmt.Println(findShort(nums2))
	nums3 := []int{1}
	fmt.Println(findShort(nums3))
	nums4 := []int{0, 2, 4, 7, 10, 11, 7, 12, 13, 14, 16, 19, 29}
	fmt.Println(findShort(nums4))
	nums5 := []int{2, 1}
	fmt.Println(findShort(nums5))
}