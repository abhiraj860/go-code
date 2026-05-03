// Go's golang.org/x/sync/semaphore package provides semaphore.Weighted. Call Acquire(ctx, n) to obtain n permits and Release(n) to return them. The context parameter allows cancellation of blocked acquires.

package main

import (
	"context"
	"io"
	"net/http"
	"golang.org/x/sync/semaphore"
)

type Response struct {
	Body       []byte
	StatusCode int
}

type APIClient struct {
	sem *semaphore.Weighted
}

func NewAPIClient() *APIClient {
	return &APIClient{sem: semaphore.NewWeighted(5)}
}

func (c *APIClient) MakeRequest(ctx context.Context, endpoint string) (Response, error) {
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return Response{}, err
	}
	defer c.sem.Release(1)

	resp, err := http.Get(endpoint)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	return Response{Body: data, StatusCode: resp.StatusCode}, nil
}