package morondanga

import (
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

func logRequest(t *testing.T, level zapcore.Level) []observer.LoggedEntry {
	t.Helper()
	core, logs := observer.New(level)
	s := &Service{cfg: &config.Config{HTTP: config.HttpConfig{}}, log: zap.New(core)}
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"customerPhone":"59899000001"}`))
	req.Header.Set("Cookie", "access_token=secret-token")
	rec := httptest.NewRecorder()
	handler := s.httpRequestLogger(nil)(func(c echo.Context) error {
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
// logged only when debugging.
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
