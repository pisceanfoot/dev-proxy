package logger

import (
	"fmt"
	"os"
	"strings"
)

// Level represents a logging verbosity level.
type Level int

const (
	// LevelError suppresses all runtime output; only fatal errors are shown.
	LevelError Level = iota
	// LevelInfo adds startup summary and config-reload notifications.
	LevelInfo
	// LevelDebug adds full per-request traces on top of info output.
	LevelDebug
)

// Logger writes levelled output to stderr.
type Logger struct {
	level Level
}

// Default is the package-level logger, initialised to LevelError (silent).
var Default = &Logger{level: LevelError}

// SetLevel updates the threshold on the Default logger.
func SetLevel(l Level) { Default.level = l }

// ParseLevel converts a string to a Level. Accepts "error", "info", "debug"
// (case-insensitive). Returns an error for any other value.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "error":
		return LevelError, nil
	case "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	default:
		return LevelError, fmt.Errorf("invalid log_level %q — must be one of: error, info, debug", s)
	}
}

// Info writes an info-level message if the logger threshold allows it.
func (l *Logger) Info(format string, args ...any) {
	if l.level >= LevelInfo {
		fmt.Fprintf(os.Stderr, "[info] "+format+"\n", args...)
	}
}

// Debug writes a debug-level message if the logger threshold allows it.
func (l *Logger) Debug(format string, args ...any) {
	if l.level >= LevelDebug {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

// Error writes an error-level message (always printed regardless of level).
func (l *Logger) Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[error] "+format+"\n", args...)
}

// Package-level convenience functions delegating to Default.

func Info(format string, args ...any)  { Default.Info(format, args...) }
func Debug(format string, args ...any) { Default.Debug(format, args...) }
func Error(format string, args ...any) { Default.Error(format, args...) }
