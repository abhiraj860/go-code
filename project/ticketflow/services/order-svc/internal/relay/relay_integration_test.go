package relay_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	segkafka "github.com/segmentio/kafka-go"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/postgres"
	"github.com/abhiraj860/ticketflow/pkg/testsupport"
	order "github.com/abhiraj860/ticketflow/services/order-svc"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/relay"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/repo"
)

func brokers() []string {
	if b := os.Getenv("TEST_KAFKA_BROKERS"); b != "" {
		return []string{b}
	}
	return []string{"localhost:9092"}
}

// End to end over REAL Postgres and REAL Kafka: place an order, run the relay,
// and read the message back off the topic. Everything until now proved the
// outbox row was written correctly; this proves it actually reaches the bus.
func TestRelayDeliversOrderCreatedToKafka(t *testing.T) {
	ctx := context.Background()

	admin := "postgres://ticketflow:ticketflow@localhost:5432/ticketflow?sslmode=disable"
	adminPool, err := pgxpool.New(ctx, admin)
	if err != nil {
		testsupport.SkipOrFail(t, "postgres not reachable: %v", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		testsupport.SkipOrFail(t, "postgres not reachable: %v", err)
	}

	name := fmt.Sprintf("tf_relay_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating scratch db: %v", err)
	}
	t.Cleanup(func() {
		p, _ := pgxpool.New(context.Background(), admin)
		if p != nil {
			defer p.Close()
			_, _ = p.Exec(context.Background(),
				"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1", name)
			_, _ = p.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name)
		}
	})

	dsn := fmt.Sprintf("postgres://ticketflow:ticketflow@localhost:5432/%s?sslmode=disable", name)
	if err := postgres.Migrate(dsn, order.Migrations, order.MigrationsDir); err != nil {
		t.Fatalf("migrating: %v", err)
	}
	pool, err := postgres.Connect(ctx, postgres.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	t.Cleanup(pool.Close)

	producer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{Brokers: brokers()})
	if err != nil {
		testsupport.SkipOrFail(t, "kafka producer: %v", err)
	}
	t.Cleanup(func() { _ = producer.Close() })

	r := repo.NewOrderRepo(pool)

	// Reachability check up front, so an unreachable broker fails fast instead
	// of as a confusing timeout later.
	conn, err := segkafka.DialContext(ctx, "tcp", brokers()[0])
	if err != nil {
		testsupport.SkipOrFail(t, "kafka not reachable (run `make up`): %v", err)
	}
	_ = conn.Close()

	placed, _, err := r.PlaceOrder(ctx, domain.PlaceOrderRequest{
		UserID: "u1", EventID: "evt-1", HoldID: uuid.NewString(),
		SeatIDs: []string{"S-1"}, TotalMinor: 150000, CurrencyCode: "INR",
		IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	rel := relay.New(relay.Options{
		Store: r, Publisher: producer, BatchSize: 10,
		Interval: 50 * time.Millisecond,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	runCtx, cancel := context.WithCancel(ctx)
	go rel.Run(runCtx)
	defer cancel()

	// The reader is created AFTER the order is placed, and reads from the
	// beginning of the topic, scanning for this run's unique order id.
	//
	// The obvious alternative -- a fresh consumer group starting at LastOffset
	// -- is a race: joining a group and receiving partition assignments takes
	// seconds, during which the relay has already published, so "latest"
	// resolves to a point past the very message under test. That produced a
	// failure that looked exactly like a broken relay when the relay was fine.
	reader := segkafka.NewReader(segkafka.ReaderConfig{
		Brokers:     brokers(),
		Topic:       tfkafka.TopicOrderCreated,
		GroupID:     "relay-test-" + uuid.NewString(),
		StartOffset: segkafka.FirstOffset,
		MaxWait:     200 * time.Millisecond,
	})
	t.Cleanup(func() { _ = reader.Close() })

	readCtx, readCancel := context.WithTimeout(ctx, 30*time.Second)
	defer readCancel()

	// The relay may republish unrelated rows; scan for ours.
	deadline := time.Now().Add(30 * time.Second)
	var found bool
	for time.Now().Before(deadline) && !found {
		msg, err := reader.ReadMessage(readCtx)
		if err != nil {
			break
		}
		env, err := tfkafka.Unmarshal[tfkafka.OrderCreated](msg.Value)
		if err != nil {
			continue
		}
		if env.Payload.OrderID == placed.ID {
			found = true
			if string(msg.Key) != placed.ID {
				t.Errorf("partition key = %q, want the order id %q", msg.Key, placed.ID)
			}
			if env.Payload.TotalMinor != 150000 {
				t.Errorf("total = %d, want 150000", env.Payload.TotalMinor)
			}
			if env.ID == "" {
				t.Error("envelope carries no id; consumers cannot dedup")
			}
		}
	}

	if !found {
		t.Fatal("the order.created message never arrived on Kafka")
	}

	// And the backlog must have drained.
	for range 40 {
		if n, _ := r.PendingCount(ctx); n == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	n, _ := r.PendingCount(ctx)
	t.Errorf("outbox backlog = %d after delivery, want 0", n)
}
