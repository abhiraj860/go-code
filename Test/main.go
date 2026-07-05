package main

import "fmt"
func firstFn(nums []int, target int) int {
    low := 0
    high := len(nums) - 1
    ans := -1
    for low <= high {
        mid := low + (high - low) / 2
        if(nums[mid] == target) {
            high = mid - 1
            ans = mid
        } else if nums[mid] < target {
            low = mid + 1
        } else {
            high = mid - 1
        }
    }
    return ans
}

func secondFn(nums []int, target int) int {
    low := 0
    high := len(nums) - 1
    ans := -1
    for low <= high {
        mid := low + (high - low) / 2
        if (nums[mid] == target) {
            ans = mid
            low = mid + 1
        } else if nums[mid] < target {
            low = mid + 1
        } else {
            high = mid - 1
        }
    }
    return ans
}


func main() {
	nums := []int{}
	target := 0
	first := firstFn(nums, target)
    last := secondFn(nums, target)
    fmt.Println(first, last)
}