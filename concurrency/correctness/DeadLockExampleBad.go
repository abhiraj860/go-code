package correctness

import "sync"

type TicketBookingDeadlock struct {
	seatLocks map[string]*sync.Mutex
}

func (tb *TicketBookingDeadlock) getLock(seatId string) *sync.Mutex {
	return tb.seatLocks[seatId]
}

// BROKEN: Can deadlock if two goroutines swap in opposite order
func (tb *TicketBookingDeadlock) SwapSeats(visitor1, seat1, visitor2, seat2 string) bool {
	tb.getLock(seat1).Lock()
	defer tb.getLock(seat1).Unlock()

	tb.getLock(seat2).Lock()
	defer tb.getLock(seat2).Unlock()

	// ... perform swap
	return true
}
