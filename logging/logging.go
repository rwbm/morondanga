package logging

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

var (
	loggerMu sync.RWMutex
	logger   *zap.Logger
	cfgCache = struct {
		level  int
		format string
	}{
		level:  int(zapcore.InfoLevel),
		format: "json",
	}
)

// Get returns a logger instance.
func Get() *zap.Logger {
	loggerMu.RLock()
	current := logger
	loggerMu.RUnlock()
	if current != nil {
		return current
	}

	loggerMu.Lock()
	defer loggerMu.Unlock()

	if logger != nil {
		return logger
	}

	logger = configLogger(cfgCache.level, cfgCache.format)
	return logger
}

// GetWithConfig returns a logger instance.
func GetWithConfig(level int, format string) *zap.Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	cfgCache.level = level
	if format == "" {
		format = cfgCache.format
	}
	cfgCache.format = format

	logger = configLogger(level, cfgCache.format)
	return logger
}

const (
	traceIDFieldName             = "trace_id"
	traceIDContextKey contextKey = traceIDFieldName
	loggerContextKey  contextKey = "request_logger"
)

// ContextWithTraceID returns a new context containing the provided trace identifier.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

// TraceIDFromContext extracts the trace identifier from the context.
func TraceIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	traceID, ok := ctx.Value(traceIDContextKey).(string)
	if !ok || traceID == "" {
		return "", false
	}

	return traceID, true
}

// WithContext returns a logger enriched with values derived from the context.
func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return Get()
	}

	if l, ok := LoggerFromContext(ctx); ok {
		return l
	}

	l := Get()

	if traceID, ok := TraceIDFromContext(ctx); ok {
		return l.With(zap.String(traceIDFieldName, traceID))
	}

	return l
}

// ContextWithLogger attaches a logger to the context so it can be reused downstream.
func ContextWithLogger(ctx context.Context, l *zap.Logger) context.Context {
	if ctx == nil || l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey, l)
}

// LoggerFromContext retrieves a logger previously attached to the context.
func LoggerFromContext(ctx context.Context) (*zap.Logger, bool) {
	if ctx == nil {
		return nil, false
	}

	l, ok := ctx.Value(loggerContextKey).(*zap.Logger)
	if !ok || l == nil {
		return nil, false
	}

	return l, true
}

func configLogger(level int, format string) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapcore.Level(level)),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          format,
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}
	return zap.Must(config.Build())
}
