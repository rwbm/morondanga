package logging

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetWithConfigCachesLogger(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)
	level := int(zapcore.DebugLevel)
	l1 := GetWithConfig(level, "console")
	require.NotNil(t, l1)

	l2 := Get()
	assert.Same(t, l1, l2)

	loggerMu.RLock()
	assert.Equal(t, level, cfgCache.level)
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

// With the OTEL bridge on, the logger must keep the configured level: what is
// below it is neither written nor exported, and a check for debug (as the
// request logger does before logging bodies) says no.
func TestTheOTELBridgeKeepsTheConfiguredLevel(t *testing.T) {
	ResetForTests()
	t.Cleanup(ResetForTests)
	previous := global.GetLoggerProvider()
	exported := &recordingExporter{}
	global.SetLoggerProvider(sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(exported))))
	t.Cleanup(func() { global.SetLoggerProvider(previous) })

	l := GetWithConfig(int(zapcore.InfoLevel), "json", OTELOptions{Enabled: true, ServiceName: "test"})

	assert.False(t, l.Core().Enabled(zapcore.DebugLevel))
	l.Debug("debug line")
	l.Info("info line")
	require.Len(t, exported.bodies(), 1)
	assert.Equal(t, "info line", exported.bodies()[0])
}

type recordingExporter struct {
	mu      sync.Mutex
	records []string
}

func (e *recordingExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range records {
		e.records = append(e.records, r.Body().AsString())
	}
	return nil
}

func (e *recordingExporter) Shutdown(context.Context) error   { return nil }
func (e *recordingExporter) ForceFlush(context.Context) error { return nil }

func (e *recordingExporter) bodies() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.records...)
}
