// Before starting a download, acquire a slot. If 3 downloads are running, the thread blocks until one finishes. The finally block ensures the slot is released when the download completes or fails. This caps concurrent downloads without needing a fixed pool of worker threads.
// Semaphores solve this directly.

package main

import (
	"context"
	"golang.org/x/sync/semaphore"
	"io"
	"net/http"
	"os"
)

type DownloadManager struct {
	sem *semaphore.Weighted
}

func NewDownloadManager() *DownloadManager {
	return &DownloadManager{sem: semaphore.NewWeighted(3)}
}

func (dm *DownloadManager) Download(ctx context.Context, url, destination string) error {
	if err := dm.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer dm.sem.Release(1)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0644)
}
