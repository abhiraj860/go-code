// Go's sync/atomic package provides atomic operations on primitive types. atomic.AddInt64() atomically adds to the value and returns the new result. Note that you must use pointers (&tb.count) and the correct type-specific function. Go also offers atomic.LoadInt64() and atomic.StoreInt64() for atomic reads and writes.

package correctness

import "sync/atomic"

type BookingStats struct {
	bookedCount int64
}

func (bs *BookingStats) OnSeatBooked() {
	atomic.AddInt64(&bs.bookedCount, 1)
}

func (bs *BookingStats) GetBookedCount() int64 {
	return atomic.LoadInt64(&bs.bookedCount)
}
