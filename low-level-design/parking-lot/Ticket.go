package parkinglot

type VehicleType int

const (
	MOTORCYCLE VehicleType = iota
	CAR
	LARGE
)

type Ticket struct {
	ID          string
	SpotID      string
	VehicleType VehicleType
	EntryTime   int64
}

func NewTicket(id string, spotID string, vehicleType VehicleType, entryTime int64) Ticket {
	return Ticket{
		ID:          id,
		SpotID:      spotID,
		VehicleType: vehicleType,
		EntryTime:   entryTime,
	}
}
