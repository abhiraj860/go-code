package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx := context.WithValue(context.Background(), "userId", "Abhiraj")
	go getValue(ctx)
	time.Sleep(10 * time.Second)
}

func getValue(ctx context.Context) {
	userId := ctx.Value("userId")
	fmt.Println("UserID", userId)
}