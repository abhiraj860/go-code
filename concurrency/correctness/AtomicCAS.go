// Go's atomic.CompareAndSwapInt64(addr, old, new) returns true if the swap succeeded. The pattern is the same: load the current value with atomic.LoadInt64(), compute the new value, attempt the CAS, and retry on failure. Go's atomics work on pointers to values, not the values directly.

// Atomics only work for single variables. The moment you need to keep two pieces of state consistent with each other, atomics can't help you.

package correctness

import "sync/atomic"

type ConcurrencyTracker struct {
	maxConcurrent int64
}

func (ct *ConcurrencyTracker) UpdateMaxConcurrent(current int64) {
	for {
		observed := atomic.LoadInt64(&ct.maxConcurrent)
		if current <= observed {
			return
		}
		if atomic.CompareAndSwapInt64(&ct.maxConcurrent, observed, current) {
			return
		}
		// CAS failed - another goroutine changed it, retry
	}
}
