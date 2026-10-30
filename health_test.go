package morondanga

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func getHealth(t *testing.T, s *Service) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	s.server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	return rec.Code, rec.Body.String()
}

func TestTheHealthCheckSaysWhichBuildIsRunning(t *testing.T) {
	s := &Service{server: echo.New()}
	s.setHealthCheck()
	s.WithVersion("913086b2a0e4", "2026-10-30T19:40:00Z")

	code, body := getHealth(t, s)

	require.Equal(t, http.StatusOK, code)
	require.JSONEq(t, `{"Status":"OK","Version":"913086b2a0e4","BuiltAt":"2026-10-30T19:40:00Z"}`, body)
}

func TestTheHealthCheckWithoutAVersionOnlySaysItIsUp(t *testing.T) {
	s := &Service{server: echo.New()}
	s.setHealthCheck()

	_, body := getHealth(t, s)

	require.JSONEq(t, `{"Status":"OK"}`, body)
}

func TestACustomHealthCheckIsServed(t *testing.T) {
	s := &Service{server: echo.New()}
	s.setHealthCheck()
	s.WithHttpHealthCheck("/health", func(c echo.Context) error {
		return c.String(http.StatusServiceUnavailable, "database down")
	})

	code, body := getHealth(t, s)

	require.Equal(t, http.StatusServiceUnavailable, code)
	require.Equal(t, "database down", body)
}
