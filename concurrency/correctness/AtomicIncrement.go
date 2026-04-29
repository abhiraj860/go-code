package correctness

import "sync/atomic"

type RequestCounter struct {
	requestCount int64
}

func (rc *RequestCounter) OnRequest() {
	atomic.AddInt64(&rc.requestCount, 1)
}
