// Package sqsconsumer polls an SQS queue and dispatches to a handler.
package sqsconsumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Handler processes one message. Returning nil deletes it from the queue;
// returning an error leaves it, so SQS redelivers after the visibility timeout
// and eventually moves it to the configured dead-letter queue.
//
// Handlers MUST be idempotent: SQS is at-least-once, and a handler that
// succeeds but whose delete call fails will see the message again.
type Handler func(ctx context.Context, body string) error

// Consumer long-polls SQS.
type Consumer struct {
	client   *sqs.Client
	queueURL string
	logger   *slog.Logger

	concurrency int
	waitSeconds int32
	batchSize   int32

	received, processed, failed atomic.Uint64
}

type Options struct {
	Client   *sqs.Client
	QueueURL string
	Logger   *slog.Logger
	// Concurrency is how many messages from one batch are handled in parallel.
	Concurrency int
	// BatchSize caps a single receive. SQS allows at most 10.
	BatchSize int32
}

func New(opts Options) (*Consumer, error) {
	switch {
	case opts.Client == nil:
		return nil, errors.New("sqsconsumer: client is required")
	case opts.QueueURL == "":
		return nil, errors.New("sqsconsumer: queue url is required")
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 4
	}
	if opts.BatchSize <= 0 || opts.BatchSize > 10 {
		opts.BatchSize = 10
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Consumer{
		client: opts.Client, queueURL: opts.QueueURL, logger: opts.Logger,
		concurrency: opts.Concurrency, batchSize: opts.BatchSize,
		// Long polling. With WaitTimeSeconds at 0 the consumer would spin,
		// burning a request per empty poll -- SQS bills per request, so a quiet
		// queue would cost more than a busy one.
		waitSeconds: 20,
	}, nil
}

// Run polls until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	c.logger.Info("sqs consumer starting",
		slog.String("queue", c.queueURL), slog.Int("concurrency", c.concurrency))

	for {
		if ctx.Err() != nil {
			c.logger.Info("sqs consumer stopping")
			return nil
		}

		out, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: c.batchSize,
			WaitTimeSeconds:     c.waitSeconds,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// A transient AWS error must not kill the consumer; back off and
			// keep polling.
			c.logger.Error("receiving from sqs failed", slog.Any("error", err))
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return nil
			}
			continue
		}

		if len(out.Messages) == 0 {
			continue
		}
		c.received.Add(uint64(len(out.Messages)))
		c.handleBatch(ctx, out.Messages, handler)
	}
}

// handleBatch fans a batch across workers and deletes only what succeeded.
func (c *Consumer) handleBatch(ctx context.Context, msgs []sqstypes.Message, handler Handler) {
	jobs := make(chan sqstypes.Message)

	var wg sync.WaitGroup
	wg.Add(c.concurrency)
	for range c.concurrency {
		go func() {
			defer wg.Done()
			for msg := range jobs {
				if err := handler(ctx, aws.ToString(msg.Body)); err != nil {
					c.failed.Add(1)
					// Deliberately NOT deleted. Leaving it invisible until the
					// visibility timeout lapses is what gives SQS its retry,
					// and what eventually routes a poison message to the DLQ.
					c.logger.ErrorContext(ctx, "handler failed; message will be redelivered",
						slog.Any("error", err))
					continue
				}
				c.deleteMessage(ctx, msg)
				c.processed.Add(1)
			}
		}()
	}

	for _, msg := range msgs {
		select {
		case jobs <- msg:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}

func (c *Consumer) deleteMessage(ctx context.Context, msg sqstypes.Message) {
	// A fresh context: the parent may already be cancelled by a shutdown, and
	// failing to delete a message that was successfully handled means doing the
	// work twice on the next boot.
	delCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	_, err := c.client.DeleteMessage(delCtx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "deleting handled message failed; it will be redelivered",
			slog.Any("error", err))
	}
}

// Stats reports consumption counts.
type Stats struct {
	Received  uint64
	Processed uint64
	Failed    uint64
}

func (c *Consumer) Stats() Stats {
	return Stats{
		Received:  c.received.Load(),
		Processed: c.processed.Load(),
		Failed:    c.failed.Load(),
	}
}

// EnsureQueue creates the queue and its dead-letter queue when absent.
//
// The redrive policy is the important half: without a DLQ a message that always
// fails is redelivered forever, consuming the consumer's attention and never
// making progress. maxReceiveCount 3 moves it aside after three attempts.
func EnsureQueue(ctx context.Context, client *sqs.Client, name string) (queueURL string, err error) {
	dlq, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(name + "-dlq"),
	})
	if err != nil {
		return "", fmt.Errorf("sqsconsumer: creating dlq: %w", err)
	}

	attrs, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       dlq.QueueUrl,
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return "", fmt.Errorf("sqsconsumer: reading dlq arn: %w", err)
	}
	dlqARN := attrs.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]

	main, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(name),
		Attributes: map[string]string{
			"RedrivePolicy": fmt.Sprintf(
				`{"deadLetterTargetArn":"%s","maxReceiveCount":"3"}`, dlqARN),
			// Long enough for a slow email provider, short enough that a
			// crashed consumer's messages return promptly.
			"VisibilityTimeout": "30",
		},
	})
	if err != nil {
		return "", fmt.Errorf("sqsconsumer: creating queue: %w", err)
	}
	return aws.ToString(main.QueueUrl), nil
}
