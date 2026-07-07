package elevator

type Direction string

const (
	Up   Direction = "UP"
	Down Direction = "DOWN"
	Idle Direction = "IDLE"
)

type Elevator struct {
	currentFloor int
	direction    Direction
	requests     map[Request]bool
}

func NewElevator() *Elevator {
	return &Elevator{
		currentFloor: 0,
		direction:    Idle,
		requests:     make(map[Request]bool),
	}
}

func (e *Elevator) AddRequest(request Request) bool {
	if request.Floor < 0 || request.Floor > 9 {
		return false
	}
	if request.Floor == e.currentFloor {
		return true
	}
	if e.requests[request] {
		return false
	}
	e.requests[request] = true
	return true
}

func (e *Elevator) Step() {
	if len(e.requests) == 0 {
		e.direction = Idle
		return
	}

	if e.direction == Idle {
		nextFloor := e.nearestRequestFloor()
		if nextFloor > e.currentFloor {
			e.direction = Up
		} else {
			e.direction = Down
		}
	}

	pickupType := PICKUP_UP
	if e.direction == Down {
		pickupType = PICKUP_DOWN
	}
	pickupRequest := NewRequest(e.currentFloor, pickupType)
	destinationRequest := NewRequest(e.currentFloor, DESTINATION)

	if e.requests[pickupRequest] || e.requests[destinationRequest] {
		delete(e.requests, pickupRequest)
		delete(e.requests, destinationRequest)

		if len(e.requests) == 0 {
			e.direction = Idle
		}
		return
	}

	if !e.hasRequestsAhead(e.direction) {
		e.toggleDirection()
		return
	}

	if e.direction == Up {
		e.currentFloor++
	} else if e.direction == Down {
		e.currentFloor--
	}
}

func (e *Elevator) hasRequestsAhead(dir Direction) bool {
	for request := range e.requests {
		if dir == Up && request.Floor > e.currentFloor {
			return true
		}
		if dir == Down && request.Floor < e.currentFloor {
			return true
		}
	}
	return false
}

func (e *Elevator) hasRequestsAtOrBeyond(floor int, dir Direction) bool {
	for request := range e.requests {
		if dir == Up && request.Floor >= floor {
			if request.Type == PICKUP_UP || request.Type == DESTINATION {
				return true
			}
		}
		if dir == Down && request.Floor <= floor {
			if request.Type == PICKUP_DOWN || request.Type == DESTINATION {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) toggleDirection() {
	if e.direction == Up {
		e.direction = Down
	} else if e.direction == Down {
		e.direction = Up
	}
}

func (e *Elevator) nearestRequestFloor() int {
	// Find nearest request to establish initial direction (deterministic)
	var nearest *Request
	minDistance := int(^uint(0) >> 1) // MaxInt
	
	for request := range e.requests {
		req := request // capture for pointer
		distance := abs(req.Floor - e.currentFloor)
		if distance < minDistance || (distance == minDistance && (nearest == nil || req.Floor < nearest.Floor)) {
			minDistance = distance
			nearest = &req
		}
	}
	
	if nearest != nil {
		return nearest.Floor
	}
	return 0
}

func (e *Elevator) GetCurrentFloor() int {
	return e.currentFloor
}

func (e *Elevator) GetDirection() Direction {
	return e.direction
}
