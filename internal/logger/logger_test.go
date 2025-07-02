package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNoopLogger(t *testing.T) {
	logger := &NoopLogger{}

	// Test that all methods can be called without panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.Debugf("test %s", "debug")
	logger.Infof("test %s", "info")
	logger.Warnf("test %s", "warn")
	logger.Errorf("test %s", "error")

	t.Log("NoopLogger methods executed without panic")
}

func TestSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slogInstance := slog.New(handler)
	logger := &SlogLogger{logger: slogInstance}

	tests := []struct {
		name     string
		logFunc  func()
		expected string
		level    string
	}{
		{
			name:     "Debug",
			logFunc:  func() { logger.Debug("debug message") },
			expected: "debug message",
			level:    "DEBUG",
		},
		{
			name:     "Info",
			logFunc:  func() { logger.Info("info message") },
			expected: "info message",
			level:    "INFO",
		},
		{
			name:     "Warn",
			logFunc:  func() { logger.Warn("warn message") },
			expected: "warn message",
			level:    "WARN",
		},
		{
			name:     "Error",
			logFunc:  func() { logger.Error("error message") },
			expected: "error message",
			level:    "ERROR",
		},
		{
			name:     "Debugf",
			logFunc:  func() { logger.Debugf("debug %s", "formatted") },
			expected: "debug formatted",
			level:    "DEBUG",
		},
		{
			name:     "Infof",
			logFunc:  func() { logger.Infof("info %s", "formatted") },
			expected: "info formatted",
			level:    "INFO",
		},
		{
			name:     "Warnf",
			logFunc:  func() { logger.Warnf("warn %s", "formatted") },
			expected: "warn formatted",
			level:    "WARN",
		},
		{
			name:     "Errorf",
			logFunc:  func() { logger.Errorf("error %s", "formatted") },
			expected: "error formatted",
			level:    "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected log output to contain %q, got %q", tt.expected, output)
			}
			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected log output to contain level %q, got %q", tt.level, output)
			}
		})
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger(true) // debug mode

	// Verify it returns a SlogLogger
	if _, ok := logger.(*SlogLogger); !ok {
		t.Errorf("Expected NewDefaultLogger to return *SlogLogger, got %T", logger)
	}

	// Test that it works
	logger.Info("test message")
	t.Log("Default logger created and working")
}
