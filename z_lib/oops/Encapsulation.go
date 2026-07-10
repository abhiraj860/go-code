// Encapsulation: Hide state, expose behaviour. Make fields private, provide methods for access.

package main


type EncapParkingSpot struct{}

func (p *EncapParkingSpot) Occupy(vehicle *EncapVehicle) {
	_ = vehicle
}

type EncapVehicle struct {
	Type string
}

type ParkingLotGood struct {
	spots []*EncapParkingSpot
}

func NewParkingLotGood() *ParkingLotGood {
	return &ParkingLotGood{spots : []*EncapParkingSpot{}}
}

func (p *ParkingLotGood) ParkingVehicle(vehicle *EncapVehicle) bool {
	spot := p.findAvailableSpot(vehicle)
	if spot == nil {
		return false
	}
	spot.Occupy(vehicle)
	return true
}

func (p *ParkingLotGood) findAvailableSpot(vehicle *EncapVehicle) *EncapParkingSpot {
	_ = vehicle
	if len(p.spots) == 0 {
		return nil
	}
	return p.spots[0]
}

func (p *ParkingLotGood) GetSpots() []*EncapParkingSpot {
	copySpots := make([]*EncapParkingSpot, len(p.spots))
	copy(copySpots, p.spots)
	return copySpots
}