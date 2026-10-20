package morondanga

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func logRequest(t *testing.T, level zapcore.Level, httpConfig ...config.HttpConfig) []observer.LoggedEntry {
	t.Helper()
	core, logs := observer.New(level)
	s := &Service{cfg: &config.Config{HTTP: config.HttpConfig{}}, log: zap.New(core)}
	if len(httpConfig) > 0 {
		s.cfg = &config.Config{HTTP: httpConfig[0]}
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"customerPhone":"59899000001"}`))
	req.Header.Set("Cookie", "access_token=secret-token")
	rec := httptest.NewRecorder()
	handler := s.httpRequestLogger(nil)(func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, `{"customerPhone":"59899000001"}`, string(body))
		c.Response().Header().Set("Set-Cookie", "refresh_token=another-secret")
		return c.JSON(http.StatusOK, map[string]string{"deliveryAddress": "Av. 18 de Julio 1234"})
	})
	require.NoError(t, handler(echo.New().NewContext(req, rec)))
	return logs.All()
}

func fields(entry observer.LoggedEntry) map[string]any {
	return entry.ContextMap()
}

// The bodies carry the customers' phones, addresses and messages: they are
// logged by default only when debugging.
func TestBodiesAreLoggedOnlyWhenDebugging(t *testing.T) {
	for _, entry := range logRequest(t, zapcore.InfoLevel) {
		_, ok := fields(entry)["body"]
		require.False(t, ok, "%s logged its body at info", entry.Message)
	}

	debug := logRequest(t, zapcore.DebugLevel)
	require.Contains(t, fields(debug[0])["body"], "59899000001")
	require.Contains(t, fields(debug[1])["body"], "18 de Julio")
}

func TestCookiesAreNeverLogged(t *testing.T) {
	for _, entry := range logRequest(t, zapcore.DebugLevel) {
		headers := fields(entry)["headers"].(map[string]string)
		for _, name := range []string{"Cookie", "Set-Cookie"} {
			if value, ok := headers[name]; ok {
				require.Equal(t, "[REDACTED]", value, "%s logged in %s", name, entry.Message)
			}
		}
	}
}

func TestLogBodiesAtInfoWithMasking(t *testing.T) {
	entries := logRequest(t, zapcore.InfoLevel, config.HttpConfig{
		LogBodies:        true,
		MaskedBodyFields: []string{"customerPhone", "deliveryAddress"},
	})
	require.Len(t, entries, 2)
	require.Equal(t, "Incoming request", entries[0].Message)
	require.Equal(t, "Outgoing response", entries[1].Message)
	for _, entry := range entries {
		require.Equal(t, zapcore.InfoLevel, entry.Level)
		require.Contains(t, fields(entry)["body"], "[REDACTED]")
		require.NotContains(t, fields(entry)["body"], "59899000001")
		require.NotContains(t, fields(entry)["body"], "18 de Julio")
	}
}
