package main

import (
	"errors"
	// "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeSimple(t *testing.T) {
	got, err := Normalize(" Hello World   ")
	if err != nil {
		t.Fatalf("unexpected error : %v", err)
	}
	want := "hello world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeTable(t *testing.T) {
	test := []struct{
		name string
		input string
		want string
		wantErr error
	}{
		{"trim spaces", " go  ", "go", nil},
		{"lowercase", "GoLang", "golang", nil},
		{"already clean", "redis", "redis", nil},
		{"empty string", "", "", ErrEmpty},
		{"only spaces", "     ", "", ErrEmpty},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error: got %v, want %v", got, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"title":"ok"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func BenchmarkNormalize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_,_  = Normalize(" hello  world  ")
	}
}

func BenchmarkNormalizeConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := strings.TrimSpace(" Hello World   ")
		out := ""
		for _, r := range s {
			out += strings.ToLower(string(r))
		}
	}
}

func BenchmarkWithSetup(b *testing.B) {
	input := strings.Repeat("Hello World ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++{
		_, _ = Normalize(input)
	}
}


















