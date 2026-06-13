package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.WithValue(context.Background(), "userId", 12334234)
	processRequest(ctx)
}

func processRequest(ctx context.Context) {
	userId := ctx.Value("userId").(int)
	fmt.Println("The name of the user is", userId)
}