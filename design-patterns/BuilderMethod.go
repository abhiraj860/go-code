// Creational Patterns
// Builder: Use when an object has lots of optional fields or messy construction details.

package main

import "fmt"

// NOTE: Go often prefers plain structs with zero values or functional options.
// Builder can be useful when validation is required before construction.

type HTTPRequest struct {
	URL string
	Method string
	Headers map[string]string
	Body string
}

type HTTPRequestBuilder struct {
	request *HTTPRequest
}

func NewHTTPRequestBuilder() *HTTPRequestBuilder {
	return &HTTPRequestBuilder{
		request: &HTTPRequest{
			Headers: make(map[string]string),
		},
	}	
}

func (b *HTTPRequestBuilder) URL(url string) *HTTPRequestBuilder {
	b.request.URL = url
	return b
}

func (b *HTTPRequestBuilder) Method(method string) *HTTPRequestBuilder {
	b.request.Method = method
	return b
}

func (b *HTTPRequestBuilder) Headers(key, value string) *HTTPRequestBuilder {
	b.request.Headers[key] = value
	return b
}

func (b *HTTPRequestBuilder) Body(body string) *HTTPRequestBuilder {
	b.request.Body = body
	return b
}

// Validation
func (b *HTTPRequestBuilder) Build() (*HTTPRequest, error) {
	if b.request.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	return b.request, nil
} 

// func main() {
// 	request, _ := NewHTTPRequestBuilder().URL("https://api.example.com").Method("POST").Headers("Content-Type", "application/json").Body(`{"key" : "value}`).Build()
// }

