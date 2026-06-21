package main

import "fmt"

func divide(a, b int) int {
	if b == 0 {
		panic("Cannot divide by zero")
	}
	return a / b
}


func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered: ", r)
		}
	}()
	fmt.Println(divide(1 , 0))
}