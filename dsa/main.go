package main

import (
	"fmt"
)

func sparseSearch(words []string, s int, e int, key string) int {
	for s <= e {
		mid := s + (e - s) / 2
		midLeft := mid - 1
		midRight := mid + 1
		if words[mid] == "" {
			for true {
				if midLeft < s && midRight > e {
					return -1
				} else if midLeft >= s && words[midLeft] != "" {
					mid = midLeft
					break
				} else if midRight <= e && words[midRight] != "" {
					mid = midRight
					break
				}
				midLeft--
				midRight++
			}
		}
		if words[mid] == key {
			return mid
		} else if key > words[mid] {
			s = mid + 1
		} else {
			e = mid - 1
		}
	}
	return -1
}

func main() {
	words := []string{"ai", "", "", "bat", "", "", "car", "cat", "", "", "dog", "", "e"}
	k := "car"
	fmt.Println(sparseSearch(words, 0, len(words) - 1, k))
}