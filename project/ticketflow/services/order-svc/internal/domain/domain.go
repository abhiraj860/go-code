// Package domain holds order-svc's internal types.
package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("order: not found")
	ErrInvalidArgument = errors.New("order: invalid argument")
	// ErrHoldAlreadyOrdered is returned when a hold already backs an order.
	// Enforced by a unique index, so two concurrent checkouts of one hold
	// cannot both succeed.
	ErrHoldAlreadyOrdered = errors.New("order: hold already has an order")
)

type OrderStatus int16

const (
	OrderStatusUnspecified OrderStatus = 0
	OrderStatusPending     OrderStatus = 1
	OrderStatusPaid        OrderStatus = 2
	OrderStatusFailed      OrderStatus = 3
	OrderStatusCancelled   OrderStatus = 4
)

// Order is a purchase.
type Order struct {
	ID             string
	UserID         string
	EventID        string
	HoldID         string
	Status         OrderStatus
	SeatIDs        []string
	TotalMinor     int64
	CurrencyCode   string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PlaceOrderRequest asks for an order against an existing inventory hold.
type PlaceOrderRequest struct {
	UserID         string
	EventID        string
	HoldID         string
	SeatIDs        []string
	TotalMinor     int64
	CurrencyCode   string
	IdempotencyKey string
}

// OutboxRecord is one pending message, written in the same transaction as the
// state change it describes.
type OutboxRecord struct {
	ID            int64
	AggregateType string
	AggregateID   string
	Topic         string
	MessageKey    string
	Payload       []byte
	Headers       map[string]string
	CreatedAt     time.Time
	Attempts      int
}
