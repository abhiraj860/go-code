// Package grpcserver adapts the catalog service to its wire contract.
//
// This layer does nothing but translate: domain types to protobuf, domain
// errors to gRPC status codes. Keeping it that thin is what lets the domain
// model be refactored without touching a contract `buf breaking` has frozen.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/domain"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/service"
)

// Server implements catalogv1.CatalogServiceServer.
type Server struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc *service.Catalog
}

func New(svc *service.Catalog) *Server {
	return &Server{svc: svc}
}

func (s *Server) GetEvent(ctx context.Context, req *catalogv1.GetEventRequest) (*catalogv1.GetEventResponse, error) {
	event, err := s.svc.GetEvent(ctx, req.GetEventId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &catalogv1.GetEventResponse{Event: eventToProto(event)}, nil
}

func (s *Server) ListEvents(ctx context.Context, req *catalogv1.ListEventsRequest) (*catalogv1.ListEventsResponse, error) {
	page, err := s.svc.ListEvents(ctx, domain.ListFilter{
		City:      req.GetCity(),
		Kind:      domain.EventKind(req.GetKind()),
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	events := make([]*catalogv1.Event, 0, len(page.Items))
	for i := range page.Items {
		events = append(events, eventToProto(page.Items[i]))
	}
	return &catalogv1.ListEventsResponse{
		Events:        events,
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *Server) GetSeatMap(ctx context.Context, req *catalogv1.GetSeatMapRequest) (*catalogv1.GetSeatMapResponse, error) {
	seatMap, err := s.svc.GetSeatMap(ctx, req.GetSeatMapId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &catalogv1.GetSeatMapResponse{SeatMap: seatMapToProto(seatMap)}, nil
}

// toStatus maps domain errors to gRPC codes. Unmapped errors become Internal
// and deliberately do not leak their message to the caller -- a SQL error
// string can disclose schema details.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, service.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func eventToProto(e domain.Event) *catalogv1.Event {
	tiers := make([]*catalogv1.PricingTier, 0, len(e.PricingTiers))
	for _, t := range e.PricingTiers {
		tiers = append(tiers, &catalogv1.PricingTier{
			Id:   t.ID,
			Name: t.Name,
			Price: &catalogv1.Money{
				AmountMinor:  t.Price.AmountMinor,
				CurrencyCode: t.Price.CurrencyCode,
			},
		})
	}

	return &catalogv1.Event{
		Id:     e.ID,
		Title:  e.Title,
		Kind:   catalogv1.EventKind(e.Kind),
		Status: catalogv1.EventStatus(e.Status),
		Venue: &catalogv1.Venue{
			Id:          e.Venue.ID,
			Name:        e.Venue.Name,
			City:        e.Venue.City,
			CountryCode: e.Venue.CountryCode,
			Address:     e.Venue.Address,
			Latitude:    e.Venue.Latitude,
			Longitude:   e.Venue.Longitude,
		},
		StartsAt:     timestamppb.New(e.StartsAt),
		EndsAt:       timestamppb.New(e.EndsAt),
		SaleOpensAt:  timestamppb.New(e.SaleOpensAt),
		PricingTiers: tiers,
		SeatMapId:    e.SeatMapID,
		Tags:         e.Tags,
		PosterUrl:    e.PosterURL,
		Version:      e.Version,
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}
}

func seatMapToProto(m domain.SeatMap) *catalogv1.SeatMap {
	seats := make([]*catalogv1.Seat, 0, len(m.Seats))
	for _, s := range m.Seats {
		seats = append(seats, &catalogv1.Seat{
			Id:            s.ID,
			Section:       s.Section,
			Row:           s.Row,
			Number:        s.Number,
			PricingTierId: s.PricingTierID,
			X:             s.X,
			Y:             s.Y,
		})
	}
	return &catalogv1.SeatMap{
		Id:            m.ID,
		VenueId:       m.VenueID,
		Seats:         seats,
		ViewboxWidth:  m.ViewboxWidth,
		ViewboxHeight: m.ViewboxHeight,
		Version:       m.Version,
	}
}
