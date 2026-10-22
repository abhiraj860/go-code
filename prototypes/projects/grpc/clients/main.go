package main

import (
	"context"
	"io" // to detect the end of a stream
	"log"
	"time"

	"clients/shoppb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	// Credentials for the connection. insecure means no TLS —
	// fine locally, never in production.
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	// Open ONE connection and share it. It multiplexes many calls
	// over a single HTTP/2 connection, so creating one per request
	// would be a real mistake.
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// The generated client. Its methods look like ordinary Go
	// functions — that's the entire point of gRPC.
	client := shoppb.NewOrderServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ---- A normal call ----

	created, err := client.CreateOrder(ctx, &shoppb.CreateOrderRequest{
		Customer: "abhiraj",
		Total:    49.99,
	})
	if err != nil {
		log.Fatal("create:", err)
	}
	log.Println("created:", created.GetOrderId())

	// ---- Fetch it back ----

	got, err := client.GetOrder(ctx, &shoppb.GetOrderRequest{
		OrderId: created.GetOrderId(),
	})
	if err != nil {
		log.Fatal("get:", err)
	}
	log.Printf("got: %s %s $%.2f",
		got.GetOrderId(), got.GetCustomer(), got.GetTotal())

	// ---- A deliberate error ----

	_, err = client.GetOrder(ctx, &shoppb.GetOrderRequest{OrderId: "nope"})
	if err != nil {
		// status.Code extracts the typed code from the error.
		// This is how you branch on failure type — much more precise
		// than parsing an error string.
		if status.Code(err) == codes.NotFound {
			log.Println("as expected: not found")
		} else {
			log.Println("unexpected:", err)
		}
	}

	// ---- A streaming call ----

	stream, err := client.WatchOrders(ctx, &shoppb.WatchRequest{
		Customer: "abhiraj",
	})
	if err != nil {
		log.Fatal("watch:", err)
	}

	log.Println("watching...")
	for {
		// Recv blocks until the next message arrives.
		o, err := stream.Recv()

		// io.EOF is the normal "server finished" signal, not a failure.
		if err == io.EOF {
			log.Println("stream ended")
			break
		}
		if err != nil {
			log.Fatal("recv:", err)
		}
		log.Printf("  update: %s $%.2f", o.GetOrderId(), o.GetTotal())
	}
}
