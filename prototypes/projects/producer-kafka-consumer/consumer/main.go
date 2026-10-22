package main

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
)


func main() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic: "my-topic",
		GroupID: "my-group",
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("offset=%d key=%s value=%s\n headersKey=%s headersValue=%s\n", m.Offset, m.Key, m.Value, m.Headers[0].Key, m.Headers[0].Value)
	}
}