package logging

import (
	"log/slog"
	"os"
)

// LogLevel represents the logging level
type LogLevel string

// LogLevel represents the logging level
const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// InitLogger initializes the global logger with the specified level
func InitLogger(level LogLevel) {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// GetLogger returns a new logger with additional context
func GetLogger(component string) *slog.Logger {
	return slog.With("component", component)
} 