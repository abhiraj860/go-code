// When behavior varies, the better approach is to isolate that behavior into its own abstraction and compose it.

package main

type DriveTrain interface {
	Start()
}

type GasEngine struct{}

func (GasEngine) Start() {
	// gas engine startup logic
}

type ElectricMotor struct{}

func (ElectricMotor) Start() {
	// electric motor startup logic
}

type CompositionCar struct {
	drivetrain DriveTrain
} 

func NewCompositionCar(drivetrain DriveTrain) *CompositionCar {
	return &CompositionCar{drivetrain: drivetrain}
}

func (c *CompositionCar) Start() {
	c.drivetrain.Start()
}