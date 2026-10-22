package main

import (
	"fmt"
	"encoding/json" // for the size comparison at the end

	// The generated package. The path matches go_package above.
	"protobuff/shoppb"

	// proto holds Marshal and Unmarshal.
	"google.golang.org/protobuf/proto"
)

func main() {
	// Build a message. The & means "address of" — protobuf messages
	// are always used as pointers, because the generated type has
	// internal state that must not be copied.
	order := &shoppb.Order{
		OrderId:  "ord-100",
		Customer: "abhiraj",
		Total:    49.99,
		// A list of pointers, one per item.
		Items: []*shoppb.Item{
			{Sku: "widget-a", Quantity: 2},
			{Sku: "widget-b", Quantity: 1},
		},
		// The enum is a real typed constant, not a loose string —
		// so a typo won't compile.
		Status: shoppb.Status_STATUS_PENDING,
	}

	// ---- ENCODE ----

	// Marshal turns the message into compact binary bytes.
	data, err := proto.Marshal(order)
	if err != nil {
		panic(err) // panic stops the program; fine for a demo
	}
	fmt.Println("protobuf bytes:", len(data))

	// Printing them shows why this isn't human-readable —
	// %v prints any value; these are raw numbers.
	fmt.Printf("raw: %v\n", data)

	// ---- DECODE ----

	// An empty message to decode into.
	var decoded shoppb.Order

	// Unmarshal parses the bytes back. The & is required because it
	// WRITES INTO decoded, so it needs its memory address.
	if err := proto.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}

	// Generated getters are nil-safe: calling GetCustomer() on a nil
	// message returns "" instead of crashing. Prefer them over
	// direct field access.
	fmt.Println("\ndecoded:")
	fmt.Println("  order:   ", decoded.GetOrderId())
	fmt.Println("  customer:", decoded.GetCustomer())
	fmt.Println("  total:   ", decoded.GetTotal())
	fmt.Println("  status:  ", decoded.GetStatus())

	// range gives the index and the value; _ discards the index.
	for _, it := range decoded.GetItems() {
		fmt.Printf("  item:     %s x%d\n", it.GetSku(), it.GetQuantity())
	}

	// ---- SIZE COMPARISON ----

	jsonData, _ := json.Marshal(map[string]any{
		"order_id": "ord-100",
		"customer": "abhiraj",
		"total":    49.99,
		"items": []map[string]any{
			{"sku": "widget-a", "quantity": 2},
			{"sku": "widget-b", "quantity": 1},
		},
		"status": "STATUS_PENDING",
	})

	fmt.Printf("\njson:     %d bytes\n", len(jsonData))
	fmt.Printf("protobuf: %d bytes\n", len(data))
}
