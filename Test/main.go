package main

import (
	"fmt"
	"sync"
)



func main() {
	var wg sync.WaitGroup
	for i :=  range 5 {
		wg.Add(1)
		go func(id int){
			fmt.Printf("Worker id : %v \n", id)
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("All worker completed")
}

