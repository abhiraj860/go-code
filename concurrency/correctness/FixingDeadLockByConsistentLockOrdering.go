// Deadlock is fixed by consistent lock ordering

package correctness

import "sync"

type TicketBookingFixed struct {
	seatLocks map[string]*sync.Mutex
}

func (tb *TicketBookingFixed) getLock(seatId string) *sync.Mutex {
	return tb.seatLocks[seatId]
}

func (tb *TicketBookingFixed) SwapSeats(visitor1, seat1, visitor2, seat2 string) bool {
	// Always acquire locks in consistent order to prevent deadlock
	first, second := seat1, seat2
	if seat1 > seat2 {
		first, second = seat2, seat1
	}

	tb.getLock(first).Lock()
	defer tb.getLock(first).Unlock()

	tb.getLock(second).Lock()
	defer tb.getLock(second).Unlock()

	// ... perform swap
	return true
}
