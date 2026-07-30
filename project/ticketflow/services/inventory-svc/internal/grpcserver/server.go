// Package grpcserver adapts inventory to its wire contract.
package grpcserver

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/domain"
)

// Service is the behaviour the server exposes, kept as an interface so handler
// mapping can be tested without a database.
type Service interface {
	GetAvailability(ctx context.Context, eventID string, seatIDs []string) ([]domain.SeatAvailability, error)
	HoldSeats(ctx context.Context, req domain.HoldRequest) (domain.HoldResult, error)
	ReleaseHold(ctx context.Context, holdID string) (int, error)
	ConfirmHold(ctx context.Context, holdID, orderID string) ([]string, error)
}

type Server struct {
	inventoryv1.UnimplementedInventoryServiceServer
	svc Service
}

func New(svc Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) GetAvailability(ctx context.Context, req *inventoryv1.GetAvailabilityRequest) (*inventoryv1.GetAvailabilityResponse, error) {
	seats, err := s.svc.GetAvailability(ctx, req.GetEventId(), req.GetSeatIds())
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*inventoryv1.SeatAvailability, 0, len(seats))
	for _, sa := range seats {
		item := &inventoryv1.SeatAvailability{
			SeatId: sa.SeatID,
			Status: inventoryv1.SeatStatus(sa.Status),
		}
		if !sa.HoldExpiresAt.IsZero() {
			item.HoldExpiresAt = timestamppb.New(sa.HoldExpiresAt)
		}
		out = append(out, item)
	}

	return &inventoryv1.GetAvailabilityResponse{
		Seats: out,
		// Sequence lets the realtime gateway discard out-of-order WebSocket
		// frames. Phase 3 replaces this with a per-event counter; wall-clock
		// nanoseconds are monotonic enough for a single writer meanwhile.
		Sequence: time.Now().UnixNano(),
	}, nil
}

func (s *Server) HoldSeats(ctx context.Context, req *inventoryv1.HoldSeatsRequest) (*inventoryv1.HoldSeatsResponse, error) {
	result, err := s.svc.HoldSeats(ctx, domain.HoldRequest{
		EventID:        req.GetEventId(),
		SeatIDs:        req.GetSeatIds(),
		UserID:         req.GetUserId(),
		IdempotencyKey: req.GetIdempotencyKey(),
		TTL:            time.Duration(req.GetTtlSeconds()) * time.Second,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &inventoryv1.HoldSeatsResponse{
		HoldId:          result.Hold.ID,
		HeldSeatIds:     result.Hold.SeatIDs,
		RejectedSeatIds: result.RejectedSeatIDs,
		ExpiresAt:       timestamppb.New(result.Hold.ExpiresAt),
	}, nil
}

func (s *Server) ReleaseHold(ctx context.Context, req *inventoryv1.ReleaseHoldRequest) (*inventoryv1.ReleaseHoldResponse, error) {
	n, err := s.svc.ReleaseHold(ctx, req.GetHoldId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &inventoryv1.ReleaseHoldResponse{ReleasedSeatCount: int32(n)}, nil
}

func (s *Server) ConfirmHold(ctx context.Context, req *inventoryv1.ConfirmHoldRequest) (*inventoryv1.ConfirmHoldResponse, error) {
	seats, err := s.svc.ConfirmHold(ctx, req.GetHoldId(), req.GetOrderId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &inventoryv1.ConfirmHoldResponse{SoldSeatIds: seats}, nil
}

// toStatus maps domain errors to gRPC codes.
//
// The distinctions matter to callers: FailedPrecondition on an expired hold
// tells the client to re-acquire, whereas ResourceExhausted on a lost race
// tells it to pick different seats. Collapsing both to Internal would make the
// checkout flow unimplementable.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, domain.ErrHoldExpired):
		return status.Error(codes.FailedPrecondition, "hold expired")
	case errors.Is(err, domain.ErrNoSeatsAvailable):
		return status.Error(codes.ResourceExhausted, "no requested seats are available")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		// Never surface the underlying message: a SQL error can disclose schema.
		return status.Error(codes.Internal, "internal error")
	}
}
