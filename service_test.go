package morondanga

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/config"
	"github.com/rwbm/morondanga/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestServiceRunReturnsServerClosed(t *testing.T) {
	logging.ResetForTests()
	defer logging.ResetForTests()

	s := &Service{
		server: echo.New(),
		cfg: &config.Config{
			HTTP: config.HttpConfig{
				Address: "127.0.0.1:0",
			},
			App: config.AppConfig{},
		},
		log: zap.NewNop(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run()
	}()

	// Allow server to start.
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.True(t, errors.Is(err, http.ErrServerClosed))
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run to return")
	}
}

func TestServiceRunWrapsStartupError(t *testing.T) {
	logging.ResetForTests()
	defer logging.ResetForTests()

	s := &Service{
		server: echo.New(),
		cfg: &config.Config{
			HTTP: config.HttpConfig{
				Address: ":invalid",
			},
			App: config.AppConfig{},
		},
		log: zap.NewNop(),
	}

	err := s.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http server start")
	assert.False(t, errors.Is(err, http.ErrServerClosed))
}

func TestServiceShutdownClosesDatabase(t *testing.T) {
	logging.ResetForTests()
	defer logging.ResetForTests()

	sqlDB := newTrackedSQLDB(t)

	db := &gorm.DB{
		Config: &gorm.Config{
			ConnPool: sqlDB,
		},
	}

	s := &Service{
		cfg: &config.Config{},
		log: zap.NewNop(),
		db:  db,
	}

	require.NoError(t, s.Shutdown(context.Background()))

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&sqlCloseCount) > 0
	}, time.Second, 10*time.Millisecond)
}

var (
	registerDriverOnce sync.Once
	sqlCloseCount      int32
)

func newTrackedSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	registerDriverOnce.Do(func() {
		sql.Register("closer", &trackingDriver{})
	})
	atomic.StoreInt32(&sqlCloseCount, 0)

	db, err := sql.Open("closer", "")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

type trackingDriver struct{}

func (d *trackingDriver) Open(string) (driver.Conn, error) {
	return &trackingConn{}, nil
}

type trackingConn struct{}

func (c *trackingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *trackingConn) Close() error {
	atomic.AddInt32(&sqlCloseCount, 1)
	return nil
}
func (c *trackingConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }

func (c *trackingConn) Ping(ctx context.Context) error { return nil }
