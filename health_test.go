package morondanga

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

func TestTheHealthCheckIsOKOnlyWhenItsChecksPass(t *testing.T) {
	s := &Service{server: echo.New()}
	s.setHealthCheck()
	s.WithVersion("913086b2a0e4", "2026-10-30T19:40:00Z")
	queueUp := errors.New("queue unreachable")
	s.WithReadinessCheck("queue", func(context.Context) error { return queueUp })
	s.WithReadinessCheck("cache", func(context.Context) error { return nil })

	code, body := getHealth(t, s)

	require.Equal(t, http.StatusServiceUnavailable, code)
	require.JSONEq(t, `{"Status":"UNAVAILABLE","Version":"913086b2a0e4","BuiltAt":"2026-10-30T19:40:00Z","Failing":["queue"]}`, body)

	queueUp = nil
	code, _ = getHealth(t, s)
	require.Equal(t, http.StatusOK, code)
}

// A load balancer stops sending it requests while it finishes the ones it has.
func TestAServiceShuttingDownIsNotReady(t *testing.T) {
	s := &Service{server: echo.New(), cfg: &config.Config{}, log: zap.NewNop()}
	s.setHealthCheck()
	s.draining.Store(true)

	code, body := getHealth(t, s)

	require.Equal(t, http.StatusServiceUnavailable, code)
	require.JSONEq(t, `{"Status":"STOPPING"}`, body)
}

func TestShuttingDownWaitsTheDrainDelayAsNotReady(t *testing.T) {
	s := &Service{server: echo.New(), cfg: &config.Config{HTTP: config.HttpConfig{DrainDelay: 150 * time.Millisecond}}, log: zap.NewNop()}
	s.setHealthCheck()

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- s.Shutdown(context.Background()) }()
	time.Sleep(50 * time.Millisecond)
	code, _ := getHealth(t, s)
	require.Equal(t, http.StatusServiceUnavailable, code)

	require.NoError(t, <-done)
	require.GreaterOrEqual(t, time.Since(start), 150*time.Millisecond)
}
