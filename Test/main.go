package main

import "fmt"

func main() {
	arr := [3]int{1, 2, 3}
	v := []int{1,2 , 3}
	v = append(v, 4)
	a := make([]int, 4, 5)
	fmt.Println(a)
	v[0] = 10
	fmt.Println(len(v))
	fmt.Println(arr)
}