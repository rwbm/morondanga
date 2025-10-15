package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ResetForTests clears internal logger state. Intended for use in tests only.
func ResetForTests() {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = nil
	cfgCache.level = int(zapcore.InfoLevel)
	cfgCache.format = "json"
}

// OverrideLoggerForTests sets the global logger to the provided instance.
func OverrideLoggerForTests(l *zap.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}
