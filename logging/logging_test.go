package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetWithConfigCachesLogger(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)
	level := int(zapcore.DebugLevel)
	l1 := GetWithConfig(level, true, "console")
	require.NotNil(t, l1)

	l2 := Get()
	assert.Same(t, l1, l2)

	loggerMu.RLock()
	assert.Equal(t, level, cfgCache.level)
	assert.True(t, cfgCache.isDev)
	assert.Equal(t, "console", cfgCache.format)
	loggerMu.RUnlock()
}

func TestWithContextAddsTraceID(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)

	core, logs := observer.New(zapcore.DebugLevel)
	loggerMu.Lock()
	logger = zap.New(core)
	loggerMu.Unlock()

	ctx := ContextWithTraceID(context.Background(), "trace-123")
	l := WithContext(ctx)
	require.NotNil(t, l)

	l.Info("hello")
	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "hello", entries[0].Message)
	assert.Equal(t, "trace-123", entries[0].ContextMap()[traceIDFieldName])

	gotID, ok := TraceIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "trace-123", gotID)
}

func TestContextWithLoggerOverridesGlobal(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)

	custom := zap.NewNop()
	ctx := ContextWithLogger(context.Background(), custom)
	got := WithContext(ctx)
	assert.Same(t, custom, got)

	retrieved, ok := LoggerFromContext(ctx)
	assert.True(t, ok)
	assert.Same(t, custom, retrieved)
}

func TestContextWithTraceIDSkipsEmpty(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)
	base := context.Background()
	ctx := ContextWithTraceID(base, "")
	assert.Equal(t, base, ctx)
	_, ok := TraceIDFromContext(ctx)
	assert.False(t, ok)
}
