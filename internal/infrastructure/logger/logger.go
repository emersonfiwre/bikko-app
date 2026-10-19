package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var once sync.Once

// Init configures the default slog logger to emit structured JSON to stdout.
func Init() {
	once.Do(func() {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slog.SetDefault(slog.New(handler))
	})
}

// Info logs an informational message using structured attributes.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Error logs an error message using structured attributes.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Warn logs a warning message using structured attributes.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Debug logs a debug message using structured attributes.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

// WarnContext logs at warning level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}
