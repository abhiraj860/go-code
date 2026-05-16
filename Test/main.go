package main

import "fmt"

func someFunc(num string) {
	fmt.Println(num)
}


func main() {
	go someFunc("1")
	go someFunc("2")
	go someFunc("3")

	fmt.Println("hi")
}