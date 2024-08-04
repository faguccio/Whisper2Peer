package testlog

import (
	"context"
	"log/slog"
)

// TestHandler filters logs by the exact level.
type TestHandler struct {
	handler slog.Handler
	level   slog.Level
}

// NewTestHandler creates a new TestHandler.
func NewTestHandler(handler slog.Handler, level slog.Level) *TestHandler {
	return &TestHandler{
		handler: handler,
		level:   level,
	}
}

// Enabled checks if the log level is exactly the one specified.
func (h *TestHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level == h.level
}

// Handle processes the log entry if it matches the level.
func (h *TestHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.Enabled(ctx, r.Level) {
		return h.handler.Handle(ctx, r)
	}
	return nil
}

// WithAttrs returns a new handler with additional attributes.
func (h *TestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TestHandler{
		handler: h.handler.WithAttrs(attrs),
		level:   h.level,
	}
}

// WithGroup returns a new handler with an additional group.
func (h *TestHandler) WithGroup(name string) slog.Handler {
	return &TestHandler{
		handler: h.handler.WithGroup(name),
		level:   h.level,
	}
}
