package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer publishes messages.
type Producer struct {
	writer *kafka.Writer
}

// ProducerOptions configures a producer.
type ProducerOptions struct {
	// Brokers is the bootstrap list. Required.
	Brokers []string

	// RequiredAcks controls durability. Defaults to RequireAll (acks=-1):
	// every in-sync replica must acknowledge before the write is considered
	// successful. Anything weaker can lose an order that Postgres already
	// committed, which is precisely the gap the outbox exists to close.
	RequiredAcks kafka.RequiredAcks

	// BatchTimeout bounds how long a message waits to be batched with others.
	// Kept short (10ms) because the outbox relay is latency-sensitive: a buyer
	// is waiting for a ticket email.
	BatchTimeout time.Duration

	WriteTimeout time.Duration
}

func NewProducer(opts ProducerOptions) (*Producer, error) {
	if len(opts.Brokers) == 0 {
		return nil, errors.New("kafka: at least one broker is required")
	}
	if opts.RequiredAcks == 0 {
		opts.RequiredAcks = kafka.RequireAll
	}
	if opts.BatchTimeout == 0 {
		opts.BatchTimeout = 10 * time.Millisecond
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 10 * time.Second
	}

	return &Producer{
		writer: &kafka.Writer{
			Addr: kafka.TCP(opts.Brokers...),
			// Balancer hashes the message key, so all messages for one
			// aggregate land on one partition and stay ordered relative to
			// each other. Ordering across aggregates is not needed and would
			// cost throughput.
			Balancer:     &kafka.Hash{},
			RequiredAcks: opts.RequiredAcks,
			BatchTimeout: opts.BatchTimeout,
			WriteTimeout: opts.WriteTimeout,
			// Topics are created by `make topics`, never implicitly: an
			// auto-created topic gets default partitioning and replication,
			// which is never what you want in production.
			AllowAutoTopicCreation: false,
			Async:                  false,
		},
	}, nil
}

// Publish sends one message and waits for the configured acknowledgements.
//
// Synchronous by design. The outbox relay must know whether a message really
// landed before it marks the row published; an async write would let it mark
// rows sent that were still buffered in memory when the process died.
func (p *Producer) Publish(ctx context.Context, topic, key string, value []byte, headers map[string]string) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	for k, v := range headers {
		msg.Headers = append(msg.Headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka: publishing to %s: %w", topic, err)
	}
	return nil
}

// PublishBatch sends several messages in one round-trip. Used by the relay,
// which claims a batch of outbox rows at a time.
func (p *Producer) PublishBatch(ctx context.Context, msgs []Message) error {
	if len(msgs) == 0 {
		return nil
	}

	out := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		km := kafka.Message{Topic: m.Topic, Key: []byte(m.Key), Value: m.Value}
		for k, v := range m.Headers {
			km.Headers = append(km.Headers, kafka.Header{Key: k, Value: []byte(v)})
		}
		out = append(out, km)
	}

	if err := p.writer.WriteMessages(ctx, out...); err != nil {
		return fmt.Errorf("kafka: publishing batch of %d: %w", len(msgs), err)
	}
	return nil
}

// Message is a topic-addressed payload awaiting publication.
type Message struct {
	Topic   string
	Key     string
	Value   []byte
	Headers map[string]string
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
