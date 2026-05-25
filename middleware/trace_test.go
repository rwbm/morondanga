package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestTraceMiddlewareGeneratesTraceIDAndLogger(t *testing.T) {
	logging.ResetForTests()
	t.Cleanup(logging.ResetForTests)
	core, logs := observer.New(zapcore.DebugLevel)

	logging.OverrideLoggerForTests(zap.New(core))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := TraceWithConfig(TraceConfig{
		Generator: func() string {
			return "generated-trace"
		},
	})(func(c echo.Context) error {
		logger := RequestLogger(c)
		logger.Info("request handled")

		traceID, ok := logging.TraceIDFromContext(c.Request().Context())
		assert.True(t, ok)
		assert.Equal(t, "generated-trace", traceID)
		return c.NoContent(http.StatusTeapot)
	})

	err := handler(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "generated-trace", rec.Header().Get(defaultTraceHeader))

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "request handled", entries[0].Message)
	assert.Equal(t, "generated-trace", entries[0].ContextMap()["trace_id"])
}

func TestTraceMiddlewareFallbackTraceID(t *testing.T) {
	logging.ResetForTests()
	t.Cleanup(logging.ResetForTests)
	logging.OverrideLoggerForTests(zap.NewNop())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := TraceWithConfig(TraceConfig{
		Generator: func() string {
			return ""
		},
	})(func(c echo.Context) error {
		traceID, ok := logging.TraceIDFromContext(c.Request().Context())
		assert.True(t, ok)
		assert.NotEmpty(t, traceID)
		return nil
	})

	err := handler(c)
	require.NoError(t, err)
	headerValue := rec.Header().Get(defaultTraceHeader)
	assert.NotEmpty(t, headerValue)
	assert.Contains(t, headerValue, fallbackTraceIDSeparator)
}

func TestRequestLoggerFallbacksToGlobal(t *testing.T) {
	logging.ResetForTests()
	t.Cleanup(logging.ResetForTests)

	logger := RequestLogger(nil)
	assert.NotNil(t, logger)
}
