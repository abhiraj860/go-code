package elevator

type RequestType int

const (
	PICKUP_UP RequestType = iota
	PICKUP_DOWN
	DESTINATION
)

type Request struct {
	Floor int
	Type  RequestType
}

func NewRequest(floor int, requestType RequestType) Request {
	return Request{
		Floor: floor,
		Type:  requestType,
	}
}


