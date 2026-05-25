package morondanga

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
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
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
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

func TestServiceInitDatabasePostgres(t *testing.T) {
	logging.ResetForTests()
	defer logging.ResetForTests()

	dsn := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	var gotDriver string
	var gotDsn string

	originalFactory := dialectorFactory
	dialectorFactory = func(driver, dsn string) (gorm.Dialector, error) {
		gotDriver = driver
		gotDsn = dsn
		return stubDialector{name: driver}, nil
	}
	t.Cleanup(func() { dialectorFactory = originalFactory })

	s := &Service{
		cfg: &config.Config{
			Database: config.DatabaseConfig{
				Enabled:  true,
				Driver:   "postgres",
				Address:  "localhost:5432",
				User:     "user",
				Password: "pass",
				Database: "db",
			},
		},
		log: zap.NewNop(),
	}

	assert.NoError(t, s.initDatabase())
	assert.Equal(t, "postgres", gotDriver)
	assert.Equal(t, dsn, gotDsn)
	assert.NotNil(t, s.db)
}

type stubDialector struct {
	name string
}

func (d stubDialector) Name() string {
	return d.name
}

func (d stubDialector) Initialize(db *gorm.DB) error {
	return nil
}

func (d stubDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return nil
}

func (d stubDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (d stubDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return nil
}

func (d stubDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	_, _ = writer.WriteString(fmt.Sprint(v))
}

func (d stubDialector) QuoteTo(writer clause.Writer, str string) {
	_, _ = writer.WriteString(str)
}

func (d stubDialector) Explain(sql string, vars ...interface{}) string {
	return fmt.Sprintf(sql, vars...)
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
