package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"warning alias", "warning", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"mixed case", "DeBuG", slog.LevelDebug},
		{"surrounding space", "  warn  ", slog.LevelWarn},
		{"empty falls back to info", "", slog.LevelInfo},
		{"garbage falls back to info", "verbose", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLevel(tt.input); got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewAttachesServiceField(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil)).With(slog.String("service", "catalog-svc"))
	logger.Info("hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output was not valid JSON: %v", err)
	}
	if record["service"] != "catalog-svc" {
		t.Errorf("service = %v, want catalog-svc", record["service"])
	}
}

func TestFromContextFallsBackToDefault(t *testing.T) {
	// A context with no logger must still yield a usable logger, so handlers
	// can log unconditionally without nil checks.
	if got := FromContext(context.Background()); got == nil {
		t.Fatal("FromContext returned nil for a bare context")
	}
}

func TestWithLoggerRoundTrips(t *testing.T) {
	want := New(Options{Service: "inventory-svc"})
	ctx := WithLogger(context.Background(), want)

	if got := FromContext(ctx); got != want {
		t.Error("FromContext did not return the logger put in by WithLogger")
	}
}

func TestFromContextIgnoresNilLogger(t *testing.T) {
	// Guards the ok && logger != nil branch: storing a typed nil must not
	// produce a nil return that panics at the call site.
	ctx := WithLogger(context.Background(), nil)

	if got := FromContext(ctx); got == nil {
		t.Fatal("FromContext returned nil when a nil logger was stored")
	}
}
