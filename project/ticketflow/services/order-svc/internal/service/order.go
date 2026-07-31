// Package service holds order-svc's business rules.
package service

import (
	"context"
	"fmt"
	"log/slog"

	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
)

// Repo is order's data dependency.
type Repo interface {
	PlaceOrder(ctx context.Context, req domain.PlaceOrderRequest) (domain.Order, bool, error)
	GetOrder(ctx context.Context, id string) (domain.Order, error)
	MarkPaid(ctx context.Context, id string) error
}

// Orders serves the order lifecycle.
type Orders struct {
	repo      Repo
	inventory inventoryv1.InventoryServiceClient
	logger    *slog.Logger
}

type Options struct {
	Repo      Repo
	Inventory inventoryv1.InventoryServiceClient
	Logger    *slog.Logger
}

func New(opts Options) *Orders {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Orders{repo: opts.Repo, inventory: opts.Inventory, logger: opts.Logger}
}

// PlaceOrder records a purchase against an existing inventory hold.
//
// The order is created PENDING and the seats stay merely HELD. Converting them
// to SOLD happens in ConfirmPayment, and the ordering is deliberate:
//
//   - selling the seats first, then writing the order, means a crash in between
//     leaves seats sold that no order paid for -- unrecoverable without manual
//     reconciliation, because nothing records who owns them
//   - writing the order first means a crash leaves a PENDING order over seats
//     still held, which the hold TTL cleans up on its own and the buyer can
//     simply retry
//
// The second failure is the recoverable one, so it is the one to have.
func (o *Orders) PlaceOrder(ctx context.Context, req domain.PlaceOrderRequest) (domain.Order, bool, error) {
	switch {
	case req.UserID == "":
		return domain.Order{}, false, fmt.Errorf("%w: user id is required", domain.ErrInvalidArgument)
	case req.EventID == "":
		return domain.Order{}, false, fmt.Errorf("%w: event id is required", domain.ErrInvalidArgument)
	case req.HoldID == "":
		return domain.Order{}, false, fmt.Errorf("%w: hold id is required", domain.ErrInvalidArgument)
	case len(req.SeatIDs) == 0:
		return domain.Order{}, false, fmt.Errorf("%w: at least one seat is required", domain.ErrInvalidArgument)
	case req.IdempotencyKey == "":
		// Required, not optional: without it a retried checkout charges twice.
		return domain.Order{}, false, fmt.Errorf("%w: idempotency key is required", domain.ErrInvalidArgument)
	case req.TotalMinor < 0:
		return domain.Order{}, false, fmt.Errorf("%w: total cannot be negative", domain.ErrInvalidArgument)
	case req.CurrencyCode == "":
		return domain.Order{}, false, fmt.Errorf("%w: currency code is required", domain.ErrInvalidArgument)
	}

	// The repo enforces idempotency in the database and reports whether this
	// call created an order or found one, so nothing here has to infer it from
	// timestamps.
	order, replayed, err := o.repo.PlaceOrder(ctx, req)
	if err != nil {
		return domain.Order{}, false, err
	}
	if replayed {
		o.logger.InfoContext(ctx, "order replayed from idempotency key",
			slog.String("order_id", order.ID), slog.String("user_id", req.UserID))
	}
	return order, replayed, nil
}

func (o *Orders) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	if id == "" {
		return domain.Order{}, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	return o.repo.GetOrder(ctx, id)
}

// ConfirmPayment settles an order: marks it PAID and converts the hold to SOLD.
//
// Inventory is asked FIRST. If it refuses -- the hold lapsed and someone else
// took the seats -- the order must not be marked paid, because marking a
// customer as having bought seats they do not hold is the worst outcome
// available. Better to fail the payment and let them re-pick.
func (o *Orders) ConfirmPayment(ctx context.Context, orderID, paymentRef string) (domain.Order, []string, error) {
	if orderID == "" {
		return domain.Order{}, nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}

	order, err := o.repo.GetOrder(ctx, orderID)
	if err != nil {
		return domain.Order{}, nil, err
	}

	// Already settled: return the existing state rather than confirming twice.
	// Payment webhooks retry, so this path is normal.
	if order.Status == domain.OrderStatusPaid {
		return order, order.SeatIDs, nil
	}

	resp, err := o.inventory.ConfirmHold(ctx, &inventoryv1.ConfirmHoldRequest{
		HoldId:  order.HoldID,
		OrderId: order.ID,
	})
	if err != nil {
		o.logger.WarnContext(ctx, "inventory refused to confirm the hold",
			slog.String("order_id", orderID), slog.String("hold_id", order.HoldID),
			slog.Any("error", err))
		return domain.Order{}, nil, fmt.Errorf("confirming hold for order %s: %w", orderID, err)
	}

	if err := o.repo.MarkPaid(ctx, orderID); err != nil {
		// The seats ARE sold and inventory records this order as their owner,
		// so nothing is double-sold. The order row simply lags, and a retry of
		// this call fixes it -- ConfirmHold is idempotent for the same order.
		o.logger.ErrorContext(ctx, "seats sold but the order could not be marked paid; retry will reconcile",
			slog.String("order_id", orderID), slog.Any("error", err))
		return domain.Order{}, nil, err
	}

	order.Status = domain.OrderStatusPaid
	o.logger.InfoContext(ctx, "order paid",
		slog.String("order_id", orderID),
		slog.String("payment_reference", paymentRef),
		slog.Int("seats", len(resp.GetSoldSeatIds())))

	return order, resp.GetSoldSeatIds(), nil
}
