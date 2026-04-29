package toolbox

import "sync/atomic"

var counter int64

func Increment() int64 {
	return atomic.AddInt64(&counter, 1) // Thread-safe increment
}