// Each write acquires permits equal to its size, rounded up to the nearest MB. Files smaller than 1 MB still acquire at least one permit to prevent unlimited tiny writes. This permit granularity trades accuracy for simplicity. A 1.1 MB file consumes 2 permits, so you might have 98 MB on disk when you block the next write, not exactly 100 MB. If not enough permits are available, the thread blocks until ongoing writes complete and release permits.

package main

import (
	"context"
	"golang.org/x/sync/semaphore"
	"os"
)

const MB = 1024 * 1024

type DiskWriter struct {
	sem *semaphore.Weighted
}

func NewDiskWriter() *DiskWriter {
	return &DiskWriter{sem: semaphore.NewWeighted(100)} // 100 MB
}

func (dw *DiskWriter) WriteFile(ctx context.Context, data []byte, path string) error {
	permits := int64(max(1, (len(data)+MB-1)/MB))

	if err := dw.sem.Acquire(ctx, permits); err != nil {
		return err
	}
	defer dw.sem.Release(permits)

	return os.WriteFile(path, data, 0644)
}
