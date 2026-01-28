package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger defines the logging interface
type Logger interface {
	LogInfo(ctx context.Context, msg string, attrs ...any)
	LogError(ctx context.Context, msg string, err error, attrs ...any)
	LogWarning(ctx context.Context, msg string, attrs ...any)
	WithRequestID(requestID string) Logger
}

// StructuredLogger implements the Logger interface
type StructuredLogger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger
func NewLogger() Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &StructuredLogger{
		Logger: slog.New(handler),
	}
}

// WithRequestID adds a request ID to the logger context
func (l *StructuredLogger) WithRequestID(requestID string) Logger {
	return &StructuredLogger{
		Logger: l.Logger.With("request_id", requestID),
	}
}

// LogError logs an error with context
func (l *StructuredLogger) LogError(ctx context.Context, msg string, err error, attrs ...any) {
	allAttrs := append([]any{"error", err.Error()}, attrs...)
	l.Logger.ErrorContext(ctx, msg, allAttrs...)
}

// LogInfo logs an info message with context
func (l *StructuredLogger) LogInfo(ctx context.Context, msg string, attrs ...any) {
	l.Logger.InfoContext(ctx, msg, attrs...)
}

// LogWarning logs a warning message with context
func (l *StructuredLogger) LogWarning(ctx context.Context, msg string, attrs ...any) {
	l.Logger.WarnContext(ctx, msg, attrs...)
}
