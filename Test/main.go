package main

import (
	"fmt"
	"slices"
)

func main() {
	size := 10
	slice := make([]int, size)
	fmt.Printf("Slice (method1): %v \n", slice)
	slice2 := make([]int, size)
	fmt.Printf("Slice (method2): %v \n", slice2)
	slice3 := []int{}
	fmt.Printf("Slice (method3): %v \n", slice3)
	slice4 := []int{4,7, 1, 8, 3, 0, 2, 67, 34, 4, 1}
	slice4 = append(slice4, 11)
	length := len(slice4)
	fmt.Printf("Length %v \n", length)
	fmt.Printf("Slice %v\n", slice4)
	slice4 = slice4[:len(slice4) - 1]
	slice4 = slice4[1:]
	fmt.Println("Slice", slice4)
	slices.Sort(slice4)
	fmt.Println("Slice", slice4)
	slices.SortFunc(slice4, func(left, right int)int {
		return right - left
	})
	fmt.Println("Slices", slice4)
	slices.Reverse(slice4)
	fmt.Println("Slices", slice4)
}