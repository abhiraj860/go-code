package main

type DriveTrain interface {
	Start()
}

type GasEngine struct {}

func (GasEngine) Start() {

}

type ElectricMotor struct {}

func (ElectricMotor) Start() {

}

type CompositionCar struct {
	driveTrain DriveTrain
}

func NewCompositionCar(driveTrain DriveTrain) *CompositionCar {
	return &CompositionCar{driveTrain: driveTrain}
}

func (c *CompositionCar) Start() {
	c.driveTrain.Start()
}