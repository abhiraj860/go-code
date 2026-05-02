package parkinglot

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type ParkingLot struct {
	spots           []*ParkingSpot
	activeTickets   map[string]Ticket
	occupiedSpotIds map[string]bool
	hourlyRateCents int64
}

func NewParkingLot(spots []*ParkingSpot, hourlyRateCents int64) *ParkingLot {
	return &ParkingLot{
		spots:           spots,
		activeTickets:   make(map[string]Ticket),
		occupiedSpotIds: make(map[string]bool),
		hourlyRateCents: hourlyRateCents,
	}
}

func (pl *ParkingLot) Enter(vehicleType VehicleType) (Ticket, error) {
	spot := pl.findAvailableSpot(vehicleType)
	if spot == nil {
		return Ticket{}, errors.New("no available spots for vehicle type")
	}

	pl.occupiedSpotIds[spot.ID] = true

	ticketID := uuid.New().String()
	entryTime := time.Now().UnixMilli()
	ticket := NewTicket(ticketID, spot.ID, vehicleType, entryTime)

	pl.activeTickets[ticketID] = ticket

	return ticket, nil
}

func (pl *ParkingLot) Exit(ticketID string) (int64, error) {
	if ticketID == "" {
		return 0, errors.New("invalid ticket ID")
	}

	ticket, exists := pl.activeTickets[ticketID]
	if !exists {
		return 0, errors.New("ticket not found or already used")
	}

	exitTime := time.Now().UnixMilli()
	fee := pl.computeFee(ticket.EntryTime, exitTime)

	delete(pl.occupiedSpotIds, ticket.SpotID)
	delete(pl.activeTickets, ticketID)

	return fee, nil
}

func (pl *ParkingLot) findAvailableSpot(vehicleType VehicleType) *ParkingSpot {
	requiredSpotType := mapVehicleTypeToSpotType(vehicleType)

	for _, spot := range pl.spots {
		if !pl.occupiedSpotIds[spot.ID] && spot.SpotType == requiredSpotType {
			return spot
		}
	}

	return nil
}

func mapVehicleTypeToSpotType(vehicleType VehicleType) SpotType {
	switch vehicleType {
	case MOTORCYCLE:
		return SPOT_MOTORCYCLE
	case CAR:
		return SPOT_CAR
	case LARGE:
		return SPOT_LARGE
	default:
		panic("unknown vehicle type")
	}
}

func (pl *ParkingLot) computeFee(entryTime int64, exitTime int64) int64 {
	durationMillis := exitTime - entryTime
	durationHours := durationMillis / (1000 * 60 * 60)

	if durationMillis%(1000*60*60) > 0 {
		durationHours++
	}

	return durationHours * pl.hourlyRateCents
}
