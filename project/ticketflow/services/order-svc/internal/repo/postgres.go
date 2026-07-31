// Package repo holds order-svc's data access, including the transactional
// outbox.
package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
)

const pgUniqueViolation = "23505"

type OrderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{pool: pool}
}

// PlaceOrder writes the order and its outbox message in ONE transaction.
//
// This is the entire point of the outbox pattern. The alternatives both lose
// data:
//
//   - publish to Kafka inside the transaction: impossible, Kafka is not
//     enrolled in the Postgres transaction, so a rollback cannot unpublish
//   - publish after the commit: a crash in the gap leaves an order that exists
//     with no message announcing it, so no ticket is ever issued and nobody
//     finds out until the customer complains
//
// Writing the message to a table makes it atomic with the order: both land or
// neither does. A relay then delivers it. The cost is at-least-once semantics,
// which is why every consumer must be idempotent.
func (r *OrderRepo) PlaceOrder(ctx context.Context, req domain.PlaceOrderRequest) (domain.Order, error) {
	// Idempotency pre-check outside the transaction, so the common replay path
	// costs one indexed lookup rather than a transaction plus rollback.
	if existing, err := r.findByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Order{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("order: beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	order := domain.Order{
		ID:             "ord_" + uuid.NewString(),
		UserID:         req.UserID,
		EventID:        req.EventID,
		HoldID:         req.HoldID,
		Status:         domain.OrderStatusPending,
		SeatIDs:        req.SeatIDs,
		TotalMinor:     req.TotalMinor,
		CurrencyCode:   req.CurrencyCode,
		IdempotencyKey: req.IdempotencyKey,
	}

	const insertOrder = `
		INSERT INTO customer_order
			(id, user_id, event_id, hold_id, status, seat_ids,
			 total_minor, currency_code, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err = tx.QueryRow(ctx, insertOrder,
		order.ID, order.UserID, order.EventID, order.HoldID, int16(order.Status),
		order.SeatIDs, order.TotalMinor, order.CurrencyCode, order.IdempotencyKey,
	).Scan(&order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			// Two distinct races land here, and they need different answers.
			switch pgErr.ConstraintName {
			case "customer_order_idempotency_uniq":
				// A concurrent retry of the same request won. Return its order.
				_ = tx.Rollback(ctx)
				return r.findByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey)
			case "customer_order_hold_uniq":
				// Someone already checked out this hold. Not a retry -- a
				// genuine conflict the caller must be told about.
				return domain.Order{}, domain.ErrHoldAlreadyOrdered
			}
		}
		return domain.Order{}, fmt.Errorf("order: inserting order: %w", err)
	}

	// The message, in the same transaction. If this fails, the order rolls back
	// too -- which is exactly the guarantee we want.
	if err := insertOutbox(ctx, tx, order); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("order: committing: %w", err)
	}
	return order, nil
}

// insertOutbox writes the order.created message.
func insertOutbox(ctx context.Context, tx pgx.Tx, order domain.Order) error {
	envelope := tfkafka.Envelope[tfkafka.OrderCreated]{
		// A stable id generated here, not at publish time: the relay may
		// deliver this row more than once, and consumers dedup on this value.
		// Generating it during publish would give each delivery a fresh id and
		// silently defeat deduplication.
		ID:            "evt_" + uuid.NewString(),
		Type:          tfkafka.TopicOrderCreated,
		AggregateID:   order.ID,
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: tfkafka.CurrentSchemaVersion,
		Payload: tfkafka.OrderCreated{
			OrderID:      order.ID,
			UserID:       order.UserID,
			EventID:      order.EventID,
			HoldID:       order.HoldID,
			SeatIDs:      order.SeatIDs,
			TotalMinor:   order.TotalMinor,
			CurrencyCode: order.CurrencyCode,
		},
	}

	payload, err := tfkafka.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("order: marshalling outbox payload: %w", err)
	}

	const insert = `
		INSERT INTO outbox
			(aggregate_type, aggregate_id, topic, message_key, payload, headers)
		VALUES ('order', $1, $2, $3, $4, $5)`

	headers, _ := json.Marshal(map[string]string{"event_id": envelope.ID})

	// Keyed by order id so every message for one order lands on one partition
	// and stays ordered relative to the others.
	if _, err := tx.Exec(ctx, insert, order.ID, tfkafka.TopicOrderCreated, order.ID, payload, headers); err != nil {
		return fmt.Errorf("order: inserting outbox row: %w", err)
	}
	return nil
}

// ClaimOutbox locks a batch of unpublished rows for delivery.
//
// FOR UPDATE SKIP LOCKED is what lets several relay replicas run at once: each
// claims a disjoint set, and a slow publish blocks nobody. Without SKIP LOCKED
// the second relay would queue behind the first and the extra replica would add
// nothing.
//
// Note there is no cursor. Rows are selected by published_at IS NULL, never by
// "id greater than the last one I saw", because BIGSERIAL ids are assigned at
// INSERT and rows become visible at COMMIT -- a transaction that took a lower id
// can commit later, and a cursor would skip it forever. See the table comment
// in 0001_init.up.sql.
func (r *OrderRepo) ClaimOutbox(ctx context.Context, tx pgx.Tx, limit int) ([]domain.OutboxRecord, error) {
	const q = `
		SELECT id, aggregate_type, aggregate_id, topic, message_key,
		       payload, headers, created_at, attempts
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`

	rows, err := tx.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("order: claiming outbox rows: %w", err)
	}
	defer rows.Close()

	var out []domain.OutboxRecord
	for rows.Next() {
		var (
			rec         domain.OutboxRecord
			headersJSON []byte
		)
		if err := rows.Scan(&rec.ID, &rec.AggregateType, &rec.AggregateID,
			&rec.Topic, &rec.MessageKey, &rec.Payload, &headersJSON,
			&rec.CreatedAt, &rec.Attempts); err != nil {
			return nil, fmt.Errorf("order: scanning outbox row: %w", err)
		}
		if len(headersJSON) > 0 {
			_ = json.Unmarshal(headersJSON, &rec.Headers)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order: iterating outbox rows: %w", err)
	}
	return out, nil
}

// MarkPublished records successful delivery.
func (r *OrderRepo) MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	const q = `UPDATE outbox SET published_at = now() WHERE id = ANY($1)`
	if _, err := tx.Exec(ctx, q, ids); err != nil {
		return fmt.Errorf("order: marking outbox rows published: %w", err)
	}
	return nil
}

// RecordFailure increments the attempt count and stores the error, so a row
// stuck at a high attempt count is visible as the operational problem it is.
func (r *OrderRepo) RecordFailure(ctx context.Context, tx pgx.Tx, ids []int64, cause string) error {
	if len(ids) == 0 {
		return nil
	}
	const q = `UPDATE outbox SET attempts = attempts + 1, last_error = $2 WHERE id = ANY($1)`
	if _, err := tx.Exec(ctx, q, ids, cause); err != nil {
		return fmt.Errorf("order: recording outbox failure: %w", err)
	}
	return nil
}

// Begin exposes a transaction for the relay, which must claim, publish and mark
// within one.
func (r *OrderRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// PendingCount reports the outbox backlog. The single most important metric
// for this pattern: a number that climbs means the relay has stalled and
// orders are being taken that nobody is fulfilling.
func (r *OrderRepo) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("order: counting pending outbox rows: %w", err)
	}
	return n, nil
}

func (r *OrderRepo) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	const q = `
		SELECT id, user_id, event_id, hold_id, status, seat_ids,
		       total_minor, currency_code, idempotency_key, created_at, updated_at
		FROM customer_order WHERE id = $1`

	var o domain.Order
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.UserID, &o.EventID, &o.HoldID, &o.Status, &o.SeatIDs,
		&o.TotalMinor, &o.CurrencyCode, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("order: querying order %q: %w", id, err)
	}
	return o, nil
}

// MarkPaid transitions an order to PAID.
func (r *OrderRepo) MarkPaid(ctx context.Context, id string) error {
	const q = `
		UPDATE customer_order SET status = 2, updated_at = now()
		WHERE id = $1 AND status = 1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("order: marking paid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OrderRepo) findByIdempotencyKey(ctx context.Context, userID, key string) (domain.Order, error) {
	if key == "" {
		return domain.Order{}, domain.ErrNotFound
	}

	const q = `
		SELECT id, user_id, event_id, hold_id, status, seat_ids,
		       total_minor, currency_code, idempotency_key, created_at, updated_at
		FROM customer_order WHERE user_id = $1 AND idempotency_key = $2`

	var o domain.Order
	err := r.pool.QueryRow(ctx, q, userID, key).Scan(
		&o.ID, &o.UserID, &o.EventID, &o.HoldID, &o.Status, &o.SeatIDs,
		&o.TotalMinor, &o.CurrencyCode, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("order: looking up idempotency key: %w", err)
	}
	return o, nil
}
