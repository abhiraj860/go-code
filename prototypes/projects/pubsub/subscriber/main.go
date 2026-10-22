package main

import (
	"context"   // carries cancellation info between function calls
	"log"
	"os"
	"os/signal"  // catch Ctrl+C for a clean exit
	"syscall"

	"github.com/redis/go-redis/v9"
)

func main() {
	// A context that cancels when Ctrl+C is pressed.
	// defer means "run this when main() ends, whatever happens".
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	// The & means "address of" — we build an Options struct and pass
	// its address, which is what the library expects.
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	// Subscribe to two channels. A channel is just a name — Redis
	// doesn't require you to create it first.
	// The "orders.*" pattern form uses PSubscribe instead; this is
	// the exact-name version.
	sub := rdb.Subscribe(ctx, "orders", "alerts")
	defer sub.Close()

	// Subscribe is lazy. Receive forces the connection and confirms
	// Redis accepted it, so we fail loudly now rather than sitting
	// silently on a broken connection.
	if _, err := sub.Receive(ctx); err != nil {
		log.Fatal("subscribe failed: ", err)
	}
	log.Println("subscribed to orders, alerts")

	// Channel() returns a Go channel — a pipe values arrive on.
	// The library runs a background goroutine feeding it.
	ch := sub.Channel()

	// range over a channel loops forever, receiving each message as
	// it arrives, and ends when the channel closes.
	for {
		select {
			case msg, ok := <-ch:
				// ok is false once the channel closes.
				if !ok {
					log.Println("channel closed")
					return
				}
				log.Printf("[%s] %s", msg.Channel, msg.Payload)

			case <-ctx.Done():
				// Fires immediately on Ctrl+C.
				log.Println("shutting down")
				return
		}
	}

}