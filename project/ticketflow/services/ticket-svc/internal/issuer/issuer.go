// Package issuer consumes order.created and produces tickets.
package issuer

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	segkafka "github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/blob"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/generator"
)

// Issuer turns orders into tickets.
type Issuer struct {
	gen    *generator.Generator
	store  blob.Store
	pub    Publisher
	logger *slog.Logger

	// seatConcurrency bounds parallelism WITHIN one order. The Kafka consumer
	// already runs a worker pool across messages; this is a second level for
	// the seats inside a single order, so a 10-seat booking renders its PDFs
	// concurrently rather than serially.
	seatConcurrency int

	issued, skipped, failed atomic.Uint64
}

// Publisher emits ticket.issued.
type Publisher interface {
	Publish(ctx context.Context, topic, key string, value []byte, headers map[string]string) error
}

type Options struct {
	Generator       *generator.Generator
	Store           blob.Store
	Publisher       Publisher
	Logger          *slog.Logger
	SeatConcurrency int
}

func New(opts Options) *Issuer {
	if opts.SeatConcurrency <= 0 {
		opts.SeatConcurrency = 4
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Issuer{
		gen:             opts.Generator,
		store:           opts.Store,
		pub:             opts.Publisher,
		logger:          opts.Logger,
		seatConcurrency: opts.SeatConcurrency,
	}
}

// Handle processes one order.created message.
//
// IDEMPOTENCY. Delivery is at-least-once -- the outbox relay resends anything
// it published but could not mark, and Kafka replays uncommitted offsets -- so
// this runs more than once for the same order routinely, not exceptionally.
//
// Two properties make that safe:
//   - the ticket id is derived from (orderID, seatID), so a replay computes the
//     SAME id rather than minting a second ticket for one seat
//   - the blob key is derived from the ticket id, so an Exists check short
//     circuits work that has already been done
//
// Neither relies on remembering anything. A fresh replica with no state handles
// a replay correctly.
func (i *Issuer) Handle(ctx context.Context, msg segkafka.Message) error {
	env, err := tfkafka.Unmarshal[tfkafka.OrderCreated](msg.Value)
	if err != nil {
		// Unparseable now means unparseable forever; retrying only blocks the
		// partition.
		return tfkafka.Permanent(err)
	}

	order := env.Payload
	if order.OrderID == "" || len(order.SeatIDs) == 0 {
		return tfkafka.Permanent(fmt.Errorf("issuer: order %q has no seats", order.OrderID))
	}

	// Fan out across the seats of one order, fan back in via errgroup. A
	// failure on any seat fails the message, so the retry regenerates the
	// batch -- safe precisely because generation is idempotent.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(i.seatConcurrency)

	for _, seatID := range order.SeatIDs {
		g.Go(func() error {
			return i.issueSeat(gctx, order, seatID)
		})
	}

	if err := g.Wait(); err != nil {
		i.failed.Add(1)
		return err
	}
	return nil
}

// issueSeat renders and stores one seat's ticket.
func (i *Issuer) issueSeat(ctx context.Context, order tfkafka.OrderCreated, seatID string) error {
	ticketID := generator.TicketID(order.OrderID, seatID)
	key := generator.PDFKey(order.EventID, ticketID)

	// The idempotency guard. Rendering a PDF is the expensive part of this
	// service, so skipping it on a replay is worth the HEAD request.
	exists, err := i.store.Exists(ctx, key)
	if err != nil {
		// A storage failure here is transient; let the retry handle it rather
		// than risking a duplicate by assuming absence.
		return fmt.Errorf("issuer: checking %q: %w", key, err)
	}
	if exists {
		i.skipped.Add(1)
		i.logger.Debug("ticket already issued, skipping",
			slog.String("ticket_id", ticketID), slog.String("order_id", order.OrderID))
		return nil
	}

	ticket := generator.Ticket{
		ID:      ticketID,
		OrderID: order.OrderID,
		EventID: order.EventID,
		SeatID:  seatID,
		UserID:  order.UserID,
	}
	ticket.Code = i.gen.GateCode(ticket.ID, ticket.EventID, ticket.SeatID)

	pdf, err := i.gen.PDF(ticket, time.Time{}, "")
	if err != nil {
		return fmt.Errorf("issuer: rendering ticket %s: %w", ticketID, err)
	}

	if err := i.store.Put(ctx, key, pdf, "application/pdf"); err != nil {
		return fmt.Errorf("issuer: storing ticket %s: %w", ticketID, err)
	}

	i.issued.Add(1)
	i.logger.Info("ticket issued",
		slog.String("ticket_id", ticketID),
		slog.String("order_id", order.OrderID),
		slog.String("seat_id", seatID),
		slog.String("key", key))

	return i.announce(ctx, ticket, key)
}

// announce publishes ticket.issued for the notification pipeline.
//
// Best effort: the ticket exists in storage and is the durable artifact, so a
// failed announcement should not fail the message and cause every seat to be
// re-rendered. Phase 4's notification service reconciles instead.
func (i *Issuer) announce(ctx context.Context, t generator.Ticket, key string) error {
	if i.pub == nil {
		return nil
	}

	env := tfkafka.Envelope[tfkafka.TicketIssued]{
		ID:            "evt_" + t.ID,
		Type:          tfkafka.TopicTicketIssued,
		AggregateID:   t.OrderID,
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: tfkafka.CurrentSchemaVersion,
		Payload: tfkafka.TicketIssued{
			TicketID: t.ID, OrderID: t.OrderID, EventID: t.EventID,
			SeatID: t.SeatID, UserID: t.UserID, PDFKey: key,
		},
	}

	raw, err := tfkafka.Marshal(env)
	if err != nil {
		i.logger.Warn("could not marshal ticket.issued", slog.Any("error", err))
		return nil
	}
	if err := i.pub.Publish(ctx, tfkafka.TopicTicketIssued, t.OrderID, raw, nil); err != nil {
		i.logger.Warn("could not announce ticket.issued; the ticket is already stored",
			slog.String("ticket_id", t.ID), slog.Any("error", err))
	}
	return nil
}

// Stats reports issuance activity. A high Skipped count is healthy -- it means
// redeliveries are being absorbed rather than duplicating work.
type Stats struct {
	Issued  uint64
	Skipped uint64
	Failed  uint64
}

func (i *Issuer) Stats() Stats {
	return Stats{
		Issued:  i.issued.Load(),
		Skipped: i.skipped.Load(),
		Failed:  i.failed.Load(),
	}
}
