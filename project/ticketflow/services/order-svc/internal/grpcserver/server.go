// Package grpcserver adapts order-svc to its wire contract.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/order/v1"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
)

// Service is the behaviour the server exposes.
type Service interface {
	PlaceOrder(ctx context.Context, req domain.PlaceOrderRequest) (domain.Order, bool, error)
	GetOrder(ctx context.Context, id string) (domain.Order, error)
	ConfirmPayment(ctx context.Context, orderID, paymentRef string) (domain.Order, []string, error)
}

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	svc Service
}

func New(svc Service) *Server { return &Server{svc: svc} }

func (s *Server) PlaceOrder(ctx context.Context, req *orderv1.PlaceOrderRequest) (*orderv1.PlaceOrderResponse, error) {
	order, replayed, err := s.svc.PlaceOrder(ctx, domain.PlaceOrderRequest{
		UserID:         req.GetUserId(),
		EventID:        req.GetEventId(),
		HoldID:         req.GetHoldId(),
		SeatIDs:        req.GetSeatIds(),
		TotalMinor:     req.GetTotal().GetAmountMinor(),
		CurrencyCode:   req.GetTotal().GetCurrencyCode(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.PlaceOrderResponse{Order: toProto(order), Replayed: replayed}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := s.svc.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.GetOrderResponse{Order: toProto(order)}, nil
}

func (s *Server) ConfirmPayment(ctx context.Context, req *orderv1.ConfirmPaymentRequest) (*orderv1.ConfirmPaymentResponse, error) {
	order, sold, err := s.svc.ConfirmPayment(ctx, req.GetOrderId(), req.GetPaymentReference())
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.ConfirmPaymentResponse{Order: toProto(order), SoldSeatIds: sold}, nil
}

// toStatus maps domain errors to gRPC codes.
//
// ErrHoldAlreadyOrdered is AlreadyExists rather than Internal because the
// caller can act on it: the seats already belong to an order, so the right
// response is to show that order, not to retry.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, domain.ErrHoldAlreadyOrdered):
		return status.Error(codes.AlreadyExists, "this hold already has an order")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		// Never leak the underlying message: a SQL error can disclose schema.
		return status.Error(codes.Internal, "internal error")
	}
}

func toProto(o domain.Order) *orderv1.Order {
	return &orderv1.Order{
		Id:        o.ID,
		UserId:    o.UserID,
		EventId:   o.EventID,
		HoldId:    o.HoldID,
		Status:    orderv1.OrderStatus(o.Status),
		SeatIds:   o.SeatIDs,
		Total:     &orderv1.Money{AmountMinor: o.TotalMinor, CurrencyCode: o.CurrencyCode},
		CreatedAt: timestamppb.New(o.CreatedAt),
		UpdatedAt: timestamppb.New(o.UpdatedAt),
	}
}
