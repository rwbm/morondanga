package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/logging"
	"go.uber.org/zap"
)

const (
	defaultTraceHeader       = echo.HeaderXRequestID
	requestLoggerContextKey  = "request_logger"
	fallbackTraceIDSeparator = "-"
)

// Trace returns middleware that injects a trace identifier into the request context.
func Trace() echo.MiddlewareFunc {
	return TraceWithConfig(TraceConfig{})
}

// TraceConfig allows customizing how trace identifiers are sourced.
type TraceConfig struct {
	// Header defines the header checked for an existing trace identifier.
	Header string
	// Generator returns a new trace identifier when the incoming request does not provide one.
	Generator func() string
}

// TraceWithConfig returns middleware wiring the trace identifier into the request context.
func TraceWithConfig(cfg TraceConfig) echo.MiddlewareFunc {
	header := cfg.Header
	if header == "" {
		header = defaultTraceHeader
	}

	generator := cfg.Generator
	if generator == nil {
		generator = newTraceID
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traceID := c.Request().Header.Get(header)
			if traceID == "" {
				traceID = generator()
			}
			if traceID == "" {
				traceID = fallbackTraceID()
			}

			c.Response().Header().Set(header, traceID)

			ctx := logging.ContextWithTraceID(c.Request().Context(), traceID)
			requestLogger := logging.WithContext(ctx)
			ctx = logging.ContextWithLogger(ctx, requestLogger)

			req := c.Request().Clone(ctx)
			c.SetRequest(req)
			c.Set(requestLoggerContextKey, requestLogger)

			return next(c)
		}
	}
}

// RequestLogger retrieves the request-scoped logger previously stored by Trace middleware.
func RequestLogger(c echo.Context) *zap.Logger {
	if c == nil {
		return logging.Get()
	}

	if v := c.Get(requestLoggerContextKey); v != nil {
		if l, ok := v.(*zap.Logger); ok && l != nil {
			return l
		}
	}

	return logging.WithContext(c.Request().Context())
}

func newTraceID() string {
	var raw [16]byte
	if _, err := io.ReadFull(rand.Reader, raw[:]); err != nil {
		log.Printf("middleware: unable to read random bytes for trace id: %v", err)
		return fallbackTraceID()
	}
	return hex.EncodeToString(raw[:])
}

func fallbackTraceID() string {
	now := time.Now().UTC()
	return fmt.Sprintf("%d%s%d", now.UnixNano(), fallbackTraceIDSeparator, os.Getpid())
}
