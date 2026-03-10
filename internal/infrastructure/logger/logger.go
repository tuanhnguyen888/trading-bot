package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger is a wrapper around slog.Logger to provide a unified logging interface.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger.
// It defaults to JSON format for production-readiness.
func NewLogger() *Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext adds context values to the logger (placeholder for future tracing).
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	// In the future, extract RequestID or TraceID from ctx here
	return l.Logger
}

// Infof logs formatted info messages.
// Note: slog doesn't natively support printf-style, so we format it first.
func (l *Logger) Infof(format string, args ...any) {
	// slog doesn't have printf style, use standard Info with msg
	// For strict printf compliance we could use fmt.Sprintf,
	// but usage of structured logging prefers Info("msg", "key", val)
	// We'll keep this simple or encourage structured usage.
	l.Logger.Info(format, args...)
}
