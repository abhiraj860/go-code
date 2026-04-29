// Fine grained locking can cause deadlock

package correctness

import "sync"

type TicketBookingFineGrained struct {
	locksMu    sync.Mutex
	seatLocks  map[string]*sync.Mutex
	seatOwners sync.Map
}

func NewTicketBookingFineGrained() *TicketBookingFineGrained {
	return &TicketBookingFineGrained{
		seatLocks: make(map[string]*sync.Mutex),
	}
}

func (tb *TicketBookingFineGrained) getLock(seatId string) *sync.Mutex {
	tb.locksMu.Lock()
	defer tb.locksMu.Unlock()

	if _, exists := tb.seatLocks[seatId]; !exists {
		tb.seatLocks[seatId] = &sync.Mutex{}
	}
	return tb.seatLocks[seatId]
}

func (tb *TicketBookingFineGrained) BookSeat(seatId, visitorId string) bool {
	lock := tb.getLock(seatId)
	lock.Lock()
	defer lock.Unlock()

	if _, exists := tb.seatOwners.Load(seatId); exists {
		return false
	}
	tb.seatOwners.Store(seatId, visitorId)
	return true
}

