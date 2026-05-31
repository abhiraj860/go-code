package main

import "fmt"


func subarraySum(nums []int, k int) int {
	mp := make(map[int]int)
	mp[0] = 1
	sum := 0
	cnt := 0
	for _, n := range nums {
		sum += n
		remainingSum := sum - k
		cnt += mp[remainingSum]
		mp[sum]++
	}
    return cnt
}


func main() {
	nums := []int{3, 4, 7, 2, -3, 1, 4, 2}
	k := 7
	fmt.Println(subarraySum(nums, k))
	nums1 := []int{1, -1, 0}
	k1 := 0
	fmt.Println(subarraySum(nums1, k1))
}