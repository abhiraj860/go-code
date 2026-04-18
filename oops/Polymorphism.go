//Polymorphism: Let objects handle themselves. No type checking, no switch statements on types

package main

type SpotSize string

const (
	RegularSpot SpotSize = "regular"
	MotorcycleSpot SpotSize = "motorcycle"
	LargeSpot SpotSize = "large"
)

type PolyParkingSpot struct{}


type PolyVehicle interface {
	RequiredSpotSize() SpotSize
}

type CarGood struct{}

func (CarGood) RequiredSpotSize() SpotSize {
	return RegularSpot
}

type MotorcycleGood struct{}

func (MotorcycleGood) RequiredSpotSize() SpotSize {
	return MotorcycleSpot
}

type TruckGood struct{}

func (TruckGood) RequiredSpotSize() SpotSize {
	return LargeSpot
}

type ParkingLotPolyGood struct{}

func (p *ParkingLotPolyGood) ParkVehicle(vehicle PolyVehicle) bool {
	required := vehicle.RequiredSpotSize()
	spot := p.findSpotBySize(required)
	return spot != nil
}

func (p *ParkingLotPolyGood) findSpotBySize(size SpotSize) *PolyParkingSpot {
	_ = size
	return nil
}

