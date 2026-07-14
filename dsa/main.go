package main

import (
	"fmt"
)

func finalResult(str string) (int, int) {
	i := 0
	j := 0
	mp := make(map[rune]int)
	n := len(str)
	maxWindow := -1
	startIndx := 0
	for j < n {
		ch := rune(str[j])
		if _, ok := mp[ch]; ok && mp[ch] >= i {
			i = mp[ch] + 1
		}
		currWindow := j - i + 1
		mp[ch] = j
		j++
		if currWindow > maxWindow {
			startIndx = i
			maxWindow = currWindow
		}
	}
	return startIndx, maxWindow
}

func main() {
	var str string
	fmt.Scan(&str)
	a, b := finalResult(str)
	fmt.Println(str[a:a + b])
}