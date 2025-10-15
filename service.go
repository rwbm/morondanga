package morondanga

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rwbm/morondanga/config"
	"github.com/rwbm/morondanga/logging"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Represents the main component, that presents a basic set
// of modules that can be enabled or disabled by configuration.
type Service struct {
	server      *echo.Echo
	cfg         config.ConfigTemplate
	log         *zap.Logger
	db          *gorm.DB
	healthCheck func(c echo.Context) error
	jwtHandler  echo.MiddlewareFunc
}

// Sets the Validator used for the HTTP server.
func (s *Service) WithHttpValidator(v Validator) *Service {
	s.server.Validator = v
	return s
}

// Adds middleware to the chain which is run after the router.
func (s *Service) WithHttpMiddleware(middleware ...echo.MiddlewareFunc) *Service {
	s.server.Use(middleware...)
	return s
}

// Sets the health check handler.
func (s *Service) WithHttpHealthCheck(route string, f func(c echo.Context) error) *Service {
	s.healthCheck = f
	return s
}

// Returns the configuration instance.
func (s *Service) Configuration() config.ConfigTemplate {
	return s.cfg
}

// Logger instance.
func (s *Service) Log() *zap.Logger {
	return s.log
}

// Returns the database instance, which is just an instance of gorm.DB
// connected to the configured database.
func (s *Service) Database() *gorm.DB {
	return s.db
}

// Starts the service, by starting the HTTP server and all the enabled modules,
// like the database and cache connection.
//
// An error may happen for example, if the the database is miss-configured or
// unreachable. Errors happening during the startup process are not logged in and
// have to be handled by the caller.
//
// It will always return a non-nil error, which must be checked. If everything is fine
// and the server was stopped, then http.ErrServerClosed will be returned.
func (s *Service) Run() error {
	if s.Configuration().GetHTTP().JwtEnabled && s.Configuration().GetHTTP().JwtSigningKey == config.DefaultJwtSigningKey {
		s.Log().Warn("Using default jwt signing key! Please, use a different one")
	}

	err := s.server.Start(s.cfg.GetHTTP().Address)
	if errors.Is(err, http.ErrServerClosed) {
		return http.ErrServerClosed
	}
	if err != nil {
		return fmt.Errorf("http server start: %w", err)
	}
	return nil
}

// Shutdown stops the server gracefully.
func (s *Service) Shutdown(ctx context.Context) error {
	var errs []error

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
		}
	}

	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("database handle: %w", err))
		} else if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database close: %w", err))
		}
	}

	return errors.Join(errs...)
}

// NewService creates a returns a new instance of Service.
func NewService(configFilePath string) (*Service, error) {
	cfg := &config.Config{}
	return newService(configFilePath, cfg)
}

// Creates a returns a new instance of Service
// with a custom configuration type.
//
// This configuration type must follow config.ConfigTemplate
// interface in order to work, and it must also expose the fields App, HTTP, Database and Custom,
// and the custom structures, in order for the Marshall function can work properly.
func NewServiceWithCustomConfiguration(configFilePath string, cfg config.ConfigTemplate) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("the configuration template cannot be nil")
	}
	return newService(configFilePath, cfg)
}

func newService(configFilePath string, cfg config.ConfigTemplate) (*Service, error) {
	s := &Service{}

	// load configuration file
	s.cfg = cfg
	if err := s.initConfig(configFilePath, cfg); err != nil {
		return nil, err
	}

	// set logger
	s.log = logging.GetWithConfig(
		s.Configuration().GetApp().LogLevel,
		s.Configuration().GetApp().IsDevelopment,
		s.Configuration().GetApp().LogFormat)

	// configure database
	if s.Configuration().GetDatabase().Enabled {
		if err := s.initDatabase(); err != nil {
			return nil, err
		}
	}

	// configure web server
	s.initWebServer()

	return s, nil
}

func (s *Service) initConfig(cfgFile string, cfg config.ConfigTemplate) error {
	err := config.GetConfiguration(cfgFile, cfg)
	if err != nil {
		return fmt.Errorf("failed to load configuration file: %s", err)
	}

	s.Configuration().SetDefaults()
	return nil
}

func (s *Service) initDatabase() error {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gormlogger.Config{
			SlowThreshold:             time.Second,     // Slow SQL threshold
			LogLevel:                  gormlogger.Info, // Log level
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,            // Don't include params in the SQL log
			Colorful:                  true,            // Disable color
		},
	)

	connString := s.Configuration().GetDatabase().ConnectionString()
	db, err := gorm.Open(
		mysql.Open(connString),
		&gorm.Config{
			Logger: newLogger,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %w", err)
	}

	s.db = db
	return nil
}
