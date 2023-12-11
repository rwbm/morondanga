package logger

import (
	"log/slog"
	"os"
)

type Level int

// Log levels
const (
	LevelDebug Level = -1
	LevelInfo  Level = 0
	LevelWarn  Level = 1
	LevelError Level = 2
)

// NewLogger returns a new instance of the internal logger,
// which is jus a wrapper to slog.Logger.
func NewLogger(level Level) *Logger {
	return &Logger{
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

// Logger is the service's internal logger
type Logger struct {
	logger *slog.Logger
	level  Level
}

func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	if l.level >= LevelWarn {
		l.logger.Warn(msg, args...)
	}
}

func (l *Logger) Info(msg string, args ...any) {
	if l.level >= LevelInfo {
		l.logger.Info(msg, args...)
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	if l.level >= LevelDebug {
		l.logger.Debug(msg, args...)
	}
}
