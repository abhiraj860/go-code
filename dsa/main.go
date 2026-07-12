package main

import (
	"fmt"
)

func merge(nums []int, s int, mid int, e int, cnt *int) {
	i := s
	j := mid + 1
	temp := []int{}
	for i <= mid && j <= e {
		if nums[i] <= nums[j] {
			temp = append(temp, nums[i])
			i++
		} else {
			*cnt += (mid - i + 1)
			temp = append(temp, nums[j])
			j++
		}
	}
	for i <= mid {
		temp = append(temp, nums[i])
		i++
	}
	for j <= e {
		temp = append(temp, nums[j])
		j++
	}
	indx := 0
	for i := s; i <= e; i++ {
		nums[i] = temp[indx]
		indx++
	}
}

func countNums(nums []int, s int, e int, cnt *int) {
	if s >= e {
		return 
	}
	mid := (s + e) / 2
	countNums(nums, s, mid, cnt)
	countNums(nums, mid + 1, e, cnt)
	merge(nums, s, mid, e, cnt)
}

func main() {
	nums := []int{4, 3, 2, 1}
	s := 0
	e := len(nums) - 1
	cnt := 0	
	countNums(nums, s, e, &cnt)
	fmt.Println(cnt)
}