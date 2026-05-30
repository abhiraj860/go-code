package main

import (
	"fmt"
	"time"
)

type RateLimit struct {
	Limit         int
	WindowSize    int
	CurrentWindow int
	Counter       int
}

func (r *RateLimit) AllowRequest() bool {
	now := time.Now().UnixMilli() / 1000
	window := int(now) / r.WindowSize

	if window != r.CurrentWindow {
		r.CurrentWindow = window
		r.Counter = 0
	}
	if r.Counter < r.Limit {
		r.Counter = r.Counter + 1
		return true
	} else {
		return false
	}
}

func main() {
	limit := 5
	windowSize := 5
	now := time.Now().UnixMilli() / 1000
	currentWindow := int(now) / 4
	counter := 0

	rateLimit := &RateLimit{
		Limit:         limit,
		WindowSize:    windowSize,
		CurrentWindow: currentWindow,
		Counter:       counter,
	}
	for i := 0; i < 7; i++ {
		if i == 0 {
			time.Sleep(4 * time.Second)
		}
		if i == 5 {
			time.Sleep(2 * time.Second)
		}
		if rateLimit.AllowRequest() {
			fmt.Println(i, "--> Allowed")
		} else {
			fmt.Println(i, "--> Rejected")
		}
	}
}
