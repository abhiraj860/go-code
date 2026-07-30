// Package logging provides the one logger construction every TicketFlow service
// uses. Output is JSON in every environment except local development, because
// CloudWatch Logs Insights can only query structured fields.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Format selects the handler. Local dev gets human-readable text; everything
// else gets JSON so CloudWatch can index the fields.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Options configures a logger. The zero value is valid and yields an
// info-level JSON logger.
type Options struct {
	// Service name, attached to every record as "service". Lets a single
	// CloudWatch log group be filtered per service.
	Service string
	// Level accepts "debug", "info", "warn", "error" (case-insensitive).
	// Anything unrecognised falls back to info.
	Level string
	// Format defaults to FormatJSON.
	Format Format
}

// New builds the process logger. Callers should set it as the default with
// slog.SetDefault so library code picks it up too.
func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{Level: parseLevel(opts.Level)}

	var handler slog.Handler
	if opts.Format == FormatText {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	logger := slog.New(handler)
	if opts.Service != "" {
		logger = logger.With(slog.String("service", opts.Service))
	}
	return logger
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ctxKey is unexported so no other package can collide with our context keys.
type ctxKey struct{}

// WithLogger returns a context carrying logger. Request-scoped fields (request
// id, user id, trace id) get attached once at the edge and then travel with the
// context, so handlers deep in the call chain log them without threading
// parameters through every signature.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext retrieves the request-scoped logger, falling back to the default
// logger when none was attached. It never returns nil, so callers can log
// unconditionally.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}
