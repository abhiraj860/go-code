package parkinglot

type SpotType int

const (
	SPOT_MOTORCYCLE SpotType = iota
	SPOT_CAR
	SPOT_LARGE
)

type ParkingSpot struct {
	ID       string
	SpotType SpotType
}

func NewParkingSpot(id string, spotType SpotType) *ParkingSpot {
	return &ParkingSpot{
		ID:       id,
		SpotType: spotType,
	}
}
