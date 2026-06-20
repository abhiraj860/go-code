package main

import (
	"fmt"
)

func main() {
	mp1 := make(map[string]int)
	mp2 := map[string]int{"one": 1, "two":2, "three": 3}
	mp3 := map[int]string{}
	var mp4 map[string]int
	mp1["planetCount"] = 8
	mp2["fifty"] = 50
	mp3[2] = "two"
	mp3[0] = "zero"
	mp3[3] = "three"
	mp3[-1] = "negative one"
	fmt.Println(mp1)
	fmt.Println(mp2)
	fmt.Println(mp3)
	fmt.Println(mp4)
	mp5 := make(map[string]int)
	fmt.Println(mp5)
	mp4 = make(map[string]int)
	mp4["random"] = 78
	fmt.Println(mp4)
	val, found := mp1["planetCount"]
	fmt.Println(val, found)
	for key, val := range mp2{
		fmt.Println(key, val)
	}
	fmt.Println(mp3)
}