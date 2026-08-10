package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	// An empty context — a required argument carrying "stop now"
	// signals. Background() is the do-nothing version.
	ctx := context.Background()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	// Ping proves Redis is reachable before we start sending.
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("cannot reach redis: ", err)
	}

	for i := 1; i <= 100; i++ {
		// Sprintf builds a string; %d inserts a number.
		msg := fmt.Sprintf("order %d created", i)

		// Publish sends to everyone currently listening.
		// .Result() gives the subscriber count and an error.
		n, err := rdb.Publish(ctx, "orders", msg).Result()
		if err != nil {
			log.Fatal("publish: ", err)
		}

		// n is how many subscribers received it. Zero means NOBODY
		// was listening — and the message is gone forever. That's
		// the single most important thing to understand about pub/sub.
		log.Printf("published %q to %d subscribers", msg, n)

		time.Sleep(time.Second)
	}

	rdb.Publish(ctx, "alerts", "publisher finished")
	log.Println("done")
}