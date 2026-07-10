package elevator

type ElevatorController struct {
	elevators []*Elevator
}

func NewElevatorController() *ElevatorController {
	return &ElevatorController{
		elevators: []*Elevator{
			NewElevator(),
			NewElevator(),
			NewElevator(),
		},
	}
}

func (c *ElevatorController) RequestElevator(floor int, requestType RequestType) bool {
	if floor < 0 || floor > 9 {
		return false
	}
	if requestType == DESTINATION {
		return false
	}

	request := NewRequest(floor, requestType)
	best := c.selectBestElevator(request)
	if best != nil {
		return best.AddRequest(request)
	}
	return false
}

func (c *ElevatorController) Step() {
	for _, e := range c.elevators {
		e.Step()
	}
}

func (c *ElevatorController) selectBestElevator(request Request) *Elevator {
	if best := c.findCommittedToFloor(request); best != nil {
		return best
	}
	if best := c.findNearestIdle(request.Floor); best != nil {
		return best
	}
	return c.findNearest(request.Floor)
}

func (c *ElevatorController) findCommittedToFloor(request Request) *Elevator {
	floor := request.Floor
	direction := Up
	if request.Type != PICKUP_UP {
		direction = Down
	}

	var nearest *Elevator
	minDistance := int(^uint(0) >> 1)

	for _, e := range c.elevators {
		if e.GetDirection() != direction {
			continue
		}

		if (direction == Up && e.GetCurrentFloor() > floor) ||
			(direction == Down && e.GetCurrentFloor() < floor) {
			continue
		}

		if !e.hasRequestsAtOrBeyond(floor, direction) {
			continue
		}

		distance := abs(e.GetCurrentFloor() - floor)
		if distance < minDistance {
			minDistance = distance
			nearest = e
		}
	}
	return nearest
}

func (c *ElevatorController) findNearestIdle(floor int) *Elevator {
	var nearest *Elevator
	minDistance := int(^uint(0) >> 1)

	for _, e := range c.elevators {
		if e.GetDirection() != Idle {
			continue
		}

		distance := abs(e.GetCurrentFloor() - floor)
		if distance < minDistance {
			minDistance = distance
			nearest = e
		}
	}
	return nearest
}

func (c *ElevatorController) findNearest(floor int) *Elevator {
	nearest := c.elevators[0]
	minDistance := abs(c.elevators[0].GetCurrentFloor() - floor)

	for _, e := range c.elevators {
		distance := abs(e.GetCurrentFloor() - floor)
		if distance < minDistance {
			minDistance = distance
			nearest = e
		}
	}
	return nearest
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
