package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Handler processes one message. Returning an error triggers retry, then the
// dead-letter topic once attempts are exhausted.
//
// Handlers MUST be idempotent. Delivery is at-least-once: a crash between
// processing and offset commit replays the message, and so does an outbox
// relay that published before marking the row sent. Envelope.ID is stable
// across every redelivery, so it is the natural dedup key.
type Handler func(ctx context.Context, msg kafka.Message) error

// ErrPermanent marks a failure that retrying cannot fix -- a malformed payload,
// an unknown schema version, a business rule violation. Wrapping an error in it
// sends the message straight to the DLQ instead of burning the retry budget on
// something that will fail identically every time.
var ErrPermanent = errors.New("kafka: permanent failure")

// Permanent wraps err so the consumer skips retries.
func Permanent(err error) error {
	return fmt.Errorf("%w: %w", ErrPermanent, err)
}

// ConsumerOptions configures a consumer.
type ConsumerOptions struct {
	Brokers []string
	Topic   string
	// GroupID makes Kafka distribute partitions across replicas. Every replica
	// of a service must share one, or each will receive every message.
	GroupID string

	// Concurrency is the worker-pool size: how many messages are processed in
	// parallel within a batch. 1 preserves strict per-partition ordering;
	// higher trades that for throughput. Choose per consumer -- the ticket
	// generator is CPU-bound and wants many, an indexer wants few and larger
	// batches.
	Concurrency int

	// BatchSize is how many messages are claimed before the pool runs. Offsets
	// are committed once the whole batch completes, which keeps at-least-once
	// semantics simple to reason about.
	BatchSize int

	// MaxAttempts is the total tries per message before the DLQ. 1 disables
	// retry entirely.
	MaxAttempts int

	// RetryBackoff is the base delay, doubled per attempt.
	RetryBackoff time.Duration

	// DLQTopic receives messages that exhausted their retries. Empty disables
	// dead-lettering, which means a poison message blocks the partition
	// forever -- almost never what you want.
	DLQTopic string

	// DLQProducer publishes to DLQTopic. Required when DLQTopic is set.
	DLQProducer *Producer

	Logger *slog.Logger
}

// Consumer reads a topic and dispatches to a worker pool.
type Consumer struct {
	reader *kafka.Reader
	opts   ConsumerOptions
	logger *slog.Logger

	// Stats, read via Stats().
	mu                               sync.Mutex
	processed, retried, deadLettered uint64
	failed                           uint64
}

// ConsumerStats reports throughput and failure counts.
type ConsumerStats struct {
	Processed    uint64
	Retried      uint64
	DeadLettered uint64
	Failed       uint64
}

func NewConsumer(opts ConsumerOptions) (*Consumer, error) {
	switch {
	case len(opts.Brokers) == 0:
		return nil, errors.New("kafka: at least one broker is required")
	case opts.Topic == "":
		return nil, errors.New("kafka: topic is required")
	case opts.GroupID == "":
		return nil, errors.New("kafka: group id is required")
	case opts.DLQTopic != "" && opts.DLQProducer == nil:
		return nil, errors.New("kafka: DLQProducer is required when DLQTopic is set")
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 10
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.RetryBackoff <= 0 {
		opts.RetryBackoff = 100 * time.Millisecond
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: opts.Brokers,
		Topic:   opts.Topic,
		GroupID: opts.GroupID,
		// Manual commits: the offset moves only after the handler succeeds.
		// Auto-commit would advance on read, so a crash mid-processing would
		// lose the message entirely -- at-most-once, the wrong guarantee here.
		CommitInterval: 0,
		MinBytes:       1,
		MaxBytes:       10e6,
		// Bounds how long a partially-filled batch waits before running.
		MaxWait: 500 * time.Millisecond,
	})

	return &Consumer{reader: reader, opts: opts, logger: opts.Logger}, nil
}

// Run consumes until ctx is cancelled.
//
// The loop is fan-out/fan-in: one reader claims a batch sequentially, the batch
// is fanned out across Concurrency workers, results are fanned back in, and
// offsets are committed only once every message in the batch has reached a
// terminal state (processed, or dead-lettered). Committing per-message under
// concurrency would let a later offset commit before an earlier one, so a crash
// would silently skip the earlier message.
func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	c.logger.Info("consumer starting",
		slog.String("topic", c.opts.Topic),
		slog.String("group", c.opts.GroupID),
		slog.Int("concurrency", c.opts.Concurrency),
		slog.Int("batch_size", c.opts.BatchSize))

	for {
		if ctx.Err() != nil {
			c.logger.Info("consumer stopping", slog.String("topic", c.opts.Topic))
			return nil
		}

		batch, err := c.fetchBatch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if len(batch) == 0 {
			continue
		}

		c.processBatch(ctx, batch, handler)

		// Commit the whole batch at once. Safe because every message above
		// reached a terminal state.
		if err := c.reader.CommitMessages(ctx, batch...); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// The batch will be redelivered. Handlers are idempotent, so this
			// costs duplicate work rather than correctness.
			c.logger.Error("committing offsets failed; batch will be redelivered",
				slog.Any("error", err))
		}
	}
}

// fetchBatch claims up to BatchSize messages, returning early when the topic
// goes quiet so a partial batch is not held hostage waiting to fill.
func (c *Consumer) fetchBatch(ctx context.Context) ([]kafka.Message, error) {
	batch := make([]kafka.Message, 0, c.opts.BatchSize)

	// The first read blocks on the parent context: with no traffic the consumer
	// should idle here rather than spin.
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return nil, err
	}
	batch = append(batch, msg)

	// Subsequent reads use a short deadline, so a lull yields the batch we have.
	for len(batch) < c.opts.BatchSize {
		fetchCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		msg, err := c.reader.FetchMessage(fetchCtx)
		cancel()
		if err != nil {
			break
		}
		batch = append(batch, msg)
	}

	return batch, nil
}

// processBatch fans the batch out across the worker pool and waits for all of
// it.
func (c *Consumer) processBatch(ctx context.Context, batch []kafka.Message, handler Handler) {
	jobs := make(chan kafka.Message)

	var wg sync.WaitGroup
	wg.Add(c.opts.Concurrency)
	for range c.opts.Concurrency {
		go func() {
			defer wg.Done()
			for msg := range jobs {
				c.processOne(ctx, msg, handler)
			}
		}()
	}

	for _, msg := range batch {
		select {
		case jobs <- msg:
		case <-ctx.Done():
			// Shutting down: stop feeding and let workers drain what they have.
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}

// processOne runs the handler with bounded retry, then dead-letters.
func (c *Consumer) processOne(ctx context.Context, msg kafka.Message, handler Handler) {
	var lastErr error

	for attempt := 1; attempt <= c.opts.MaxAttempts; attempt++ {
		err := handler(ctx, msg)
		if err == nil {
			c.count(&c.processed)
			return
		}
		lastErr = err

		// A permanent failure will fail identically every time; skip the
		// retries and dead-letter it now.
		if errors.Is(err, ErrPermanent) {
			c.logger.Warn("permanent failure, dead-lettering immediately",
				slog.String("topic", msg.Topic),
				slog.Int64("offset", msg.Offset),
				slog.Any("error", err))
			break
		}

		if attempt < c.opts.MaxAttempts {
			c.count(&c.retried)

			// Exponential backoff. Interruptible, so shutdown is not delayed by
			// a sleeping retry.
			delay := c.opts.RetryBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}
	}

	c.count(&c.failed)
	c.deadLetter(ctx, msg, lastErr)
}

// deadLetter forwards a message that could not be processed.
//
// Without this a poison message blocks its partition indefinitely: the offset
// never advances, and every message behind it waits forever.
func (c *Consumer) deadLetter(ctx context.Context, msg kafka.Message, cause error) {
	if c.opts.DLQTopic == "" {
		c.logger.Error("message failed and no DLQ is configured; it will be dropped",
			slog.String("topic", msg.Topic),
			slog.Int64("offset", msg.Offset),
			slog.Any("error", cause))
		return
	}

	headers := map[string]string{
		// Enough context to diagnose the failure from the DLQ alone, without
		// correlating against logs that may have rotated.
		"dlq_original_topic":  msg.Topic,
		"dlq_original_offset": fmt.Sprint(msg.Offset),
		"dlq_partition":       fmt.Sprint(msg.Partition),
		"dlq_failed_at":       time.Now().UTC().Format(time.RFC3339),
		"dlq_error":           cause.Error(),
	}

	// A fresh context: the parent may already be cancelled by the shutdown
	// that caused this failure, and losing the message then would be the worst
	// possible moment.
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := c.opts.DLQProducer.Publish(sendCtx, c.opts.DLQTopic, string(msg.Key), msg.Value, headers); err != nil {
		c.logger.Error("dead-lettering failed; message will be lost",
			slog.String("topic", msg.Topic),
			slog.Int64("offset", msg.Offset),
			slog.Any("dlq_error", err),
			slog.Any("original_error", cause))
		return
	}

	c.count(&c.deadLettered)
	c.logger.Warn("message dead-lettered",
		slog.String("topic", msg.Topic),
		slog.Int64("offset", msg.Offset),
		slog.Any("error", cause))
}

func (c *Consumer) count(field *uint64) {
	c.mu.Lock()
	*field++
	c.mu.Unlock()
}

// Stats snapshots counters for the metrics endpoint.
func (c *Consumer) Stats() ConsumerStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ConsumerStats{
		Processed:    c.processed,
		Retried:      c.retried,
		DeadLettered: c.deadLettered,
		Failed:       c.failed,
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
