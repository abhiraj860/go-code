package main

import (
	"fmt"
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/google/uuid"
	"time"
)

func main() {
	w := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "my-topic",
	}

	defer w.Close()


	for i := range 100 {
		msgs := kafka.Message{
			Key: []byte(fmt.Sprintf("Key - %d", i)),
			Value: []byte(fmt.Sprintf("Hello - %d", i)),
			Headers: []kafka.Header{
				{Key: "eventId", Value: []byte(uuid.NewString())},
			},
		}
		w.WriteMessages(context.Background(), msgs)
		fmt.Println(i)
		time.Sleep(1 * time.Second)
	}

}
