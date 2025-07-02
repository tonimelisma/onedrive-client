// Package logger provides structured logging interfaces and implementations
// for the onedrive-client application. It uses Go 1.22's log/slog package
// for structured, leveled logging with proper formatting support.
package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger defines the interface for structured logging with multiple levels.
// This replaces the previous simple Debug-only interface with a comprehensive
// logging system that supports Debug, Info, Warn, and Error levels, both
// in simple and formatted variants.
type Logger interface {
	// Debug logs debug-level messages (lowest priority)
	Debug(msg string, args ...any)
	Debugf(format string, args ...any)

	// Info logs informational messages
	Info(msg string, args ...any)
	Infof(format string, args ...any)

	// Warn logs warning messages
	Warn(msg string, args ...any)
	Warnf(format string, args ...any)

	// Error logs error messages (highest priority)
	Error(msg string, args ...any)
	Errorf(format string, args ...any)
}

// NoopLogger is a logger that discards all log messages.
// It's useful for testing or when logging is completely disabled.
type NoopLogger struct{}

func (l NoopLogger) Debug(msg string, args ...any)     {}
func (l NoopLogger) Debugf(format string, args ...any) {}
func (l NoopLogger) Info(msg string, args ...any)      {}
func (l NoopLogger) Infof(format string, args ...any)  {}
func (l NoopLogger) Warn(msg string, args ...any)      {}
func (l NoopLogger) Warnf(format string, args ...any)  {}
func (l NoopLogger) Error(msg string, args ...any)     {}
func (l NoopLogger) Errorf(format string, args ...any) {}

// SlogLogger wraps Go's log/slog.Logger to implement our Logger interface.
// It provides structured logging with configurable levels and output formats.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with the specified level.
// It uses a text handler that writes to stderr for user-friendly output.
func NewSlogLogger(level slog.Level) *SlogLogger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	return &SlogLogger{
		logger: logger,
	}
}

// NewDefaultLogger creates a logger with appropriate defaults based on debug mode.
// If debug is true, it logs at Debug level; otherwise, it logs at Info level.
func NewDefaultLogger(debug bool) Logger {
	if debug {
		return NewSlogLogger(slog.LevelDebug)
	}
	return NewSlogLogger(slog.LevelInfo)
}

// Debug logs a debug-level message with optional structured attributes
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Debugf logs a debug-level message with printf-style formatting
func (l *SlogLogger) Debugf(format string, args ...any) {
	l.logger.Debug(sprintf(format, args...))
}

// Info logs an info-level message with optional structured attributes
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Infof logs an info-level message with printf-style formatting
func (l *SlogLogger) Infof(format string, args ...any) {
	l.logger.Info(sprintf(format, args...))
}

// Warn logs a warning-level message with optional structured attributes
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Warnf logs a warning-level message with printf-style formatting
func (l *SlogLogger) Warnf(format string, args ...any) {
	l.logger.Warn(sprintf(format, args...))
}

// Error logs an error-level message with optional structured attributes
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Errorf logs an error-level message with printf-style formatting
func (l *SlogLogger) Errorf(format string, args ...any) {
	l.logger.Error(sprintf(format, args...))
}

// sprintf is a helper function that safely formats strings using fmt.Sprintf
func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
