package toolbox

import (
		"golang.org/x/sync/semaphore"
		"context"
)

func CountingLocks() {
	var permits = semaphore.NewWeighted(5) // Allow 5 concurrent operations
	var ctx context.Context 
	permits.Acquire(ctx, 1) // Block if no permits available
	defer permits.Release(1) // Always release
	// doWork()
}