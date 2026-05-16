package main

import (
	"fmt"
)

func someFunc(num string) {
	fmt.Println(num)
}


func main() {
	myChannel := make(chan string)
	anotherChannel := make(chan string)

	go func() {
		myChannel <- "data"
	}()


	go func() {
		anotherChannel <- "cow"
	}()

	select {
	case msgFromMyChannel := <- myChannel:
		fmt.Println(msgFromMyChannel)
	case msgFromAnotherChannel := <- anotherChannel:
		fmt.Println(msgFromAnotherChannel)
	}
}