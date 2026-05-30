package main

import (
	"fmt"
	"time"
)

type RateLimit struct {
	Limit       int
	WindowSize  int
	PrevCount   int
	CurrCount   int
	WindowStart int
}

func (rl *RateLimit) AllowRequest() bool {
	now := time.Now().UnixMilli() / 1000
	elapsed := int(now) - rl.WindowStart

	if elapsed >= rl.WindowSize {
		rl.PrevCount = rl.CurrCount
		rl.CurrCount = 0
		rl.WindowStart = int(now)
		elapsed = 0
	}
	overlapFraction := (rl.WindowSize - elapsed) / rl.WindowSize
	estimated := rl.PrevCount*overlapFraction + rl.CurrCount
	if estimated < rl.Limit {
		rl.CurrCount = rl.CurrCount + 1
		return true
	} else {
		return false
	}
}

func main() {
	now := time.Now().UnixMilli() / 1000
	limit := 5
	windowSize := 5
	prevCount := 3
	currCount := 1
	windowStart := int(now)
	rl := &RateLimit{
		Limit:       limit,
		WindowSize:  windowSize,
		PrevCount:   prevCount,
		CurrCount:   currCount,
		WindowStart: windowStart,
	}
	for i := range 6 {
		if rl.AllowRequest() {
			fmt.Println(i, "--> Allowed")
		} else {
			fmt.Println(i, "--> Rejected")
		}
	}
}
