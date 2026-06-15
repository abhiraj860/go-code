package main

import (
	"fmt"
	"slices"
)

func canAttendMeetings(intervals [][]int)bool {
	slices.SortFunc(intervals, func (a, b []int) int {
		return a[0] - b[0]
	})
	for i, time := range intervals {
		if i < 1 {
			continue
		}
		currStart := time[0]
		prevEnd := intervals[i - 1][1] 
		if currStart < prevEnd {
			return false 
		}
	}
	return true
}

func main() {
	intervals1 := [][]int{{1, 5}, {3, 9}, {6, 8}}
	intervals2 := [][]int{{10, 12}, {6, 9}, {13, 15}}

	fmt.Println(canAttendMeetings(intervals1))
	fmt.Println(canAttendMeetings(intervals2))
}