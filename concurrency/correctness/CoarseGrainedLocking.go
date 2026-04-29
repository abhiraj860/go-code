// Go uses sync.Mutex for mutual exclusion. You call mu.Lock() to acquire and mu.Unlock() to release. The defer tb.mu.Unlock() idiom ensures the lock is released when the function returns, regardless of which return path is taken. Always defer the unlock immediately after locking to avoid forgetting to release it.

package correctness

import "sync"

type TicketBooking struct {
	mu sync.Mutex
	seatOwners map[string]string
}

func NewTicketBooking() *TicketBooking {
	return &TicketBooking{seatOwners: make(map[string]string)}
}

func (tb *TicketBooking) BookSeat(seatID, visitorID string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if _, exists := tb.seatOwners[seatID]; exists {
		return false
	}
	tb.seatOwners[seatID] = visitorID
	return true
}

// Broken: Lock	released too early
// Broken: Different lock objects