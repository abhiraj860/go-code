package main

import (
	"fmt"
	"time"
	"context"
	"math/rand"
)

func dbCall(ctx context.Context) (string, error) {
	select {
	case <-time.After(time.Duration(rand.Intn(50)) * time.Millisecond):
		return "Done", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Millisecond)
	defer cancel()
	res, err := dbCall(ctx)
	if err != nil {
		fmt.Println("The activity was timed out")
	} else {
		fmt.Printf("Activity succeeded %v\n", res)
	}


	ctx2, cancel2 := context.WithTimeout(context.Background(), 25 * time.Millisecond)
	defer cancel2()

	res2, err := dbCall(ctx2)
	if err != nil {
		fmt.Println("The activity was timed out")
	} else {
		fmt.Printf("Activity succeeded %v \n", res2)
	}
}