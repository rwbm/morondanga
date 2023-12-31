package morondanga

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rwbm/morondanga/config"
	"github.com/rwbm/morondanga/logger"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Service represents the main component, that presents a basic set
// of modules that can be enabled or disabled by configuration.
type Service struct {
	server      *echo.Echo
	cfg         config.ConfigTemplate
	log         *logger.Logger
	db          *gorm.DB
	healthCheck func(c echo.Context) error
	jwtHandler  echo.MiddlewareFunc
}

// WithHttpValidator sets the Validator used for the HTTP server.
func (s *Service) WithHttpValidator(v Validator) *Service {
	s.server.Validator = v
	return s
}

// WithHttpMiddleware adds middleware to the chain which is run after the router.
func (s *Service) WithHttpMiddleware(middleware ...echo.MiddlewareFunc) *Service {
	s.server.Use(middleware...)
	return s
}

// WithHttpHealthCheck sets the health check handler.
func (s *Service) WithHttpHealthCheck(route string, f func(c echo.Context) error) *Service {
	s.healthCheck = f
	return s
}

// Configuration returns the configuration instance.
func (s *Service) Configuration() config.ConfigTemplate {
	return s.cfg
}

// Log returns the logger instance.
func (s *Service) Log() *logger.Logger {
	return s.log
}

// Log returns the database instance, which is just an instance of gorm.DB
// connected to the configured database.
func (s *Service) Database() *gorm.DB {
	return s.db
}

// Run starts the service, by starting the HTTP server and all the enabled modules,
// like the database and cache connection.
//
// An error may happen for example, if the the database is miss-configured or
// unreachable. Errors happening during the startup process are not logged in and
// have to be handled by the caller.
//
// It will always return a non-nil error, which must be checked. If everything is fine
// and the server was stopped, then a gorm.ErrServerClosed will be returned.
func (s *Service) Run() error {
	if s.Configuration().GetHTTP().JwtEnabled && s.Configuration().GetHTTP().JwtSigningKey == config.DefaultJwtSigningKey {
		s.Log().Warn("Using default jwt signing key! Please, use a different one")
	}

	return s.server.Start(s.cfg.GetHTTP().Address)
}

// Shutdown stops the server gracefully.
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
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

// NewService creates a returns a new instance of Service.
func NewService(configFilePath string) (*Service, error) {
	cfg := &config.Config{}
	return newService(configFilePath, cfg)
}

// NewServiceWithCustomConfiguration creates a returns a new instance of Service
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
	s.log = logger.NewLogger(logger.Level(s.Configuration().GetApp().LogLevel))

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
