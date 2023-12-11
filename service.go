package morondanga

import (
	"context"
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
	cfg         *config.Config
	log         *logger.Logger
	db          *gorm.DB
	healthCheck func(c echo.Context) error
	jwtHandler  echo.MiddlewareFunc
}

// WithHttpConfig sets the http configuration, overriding the configuration
// set in the configuration file or environment variables.
func (s *Service) WithHttpConfig(httpCfg config.HttpConfig) *Service {
	s.cfg.HTTP = httpCfg
	return s
}

// WithDatabase sets the database configuration, overriding the configuration
// in the configuration file or environment variables.
func (s *Service) WithDatabase(dbCfg config.DatabaseConfig) *Service {
	s.cfg.Database = dbCfg
	return s
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
func (s *Service) Configuration() *config.Config {
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
	return s.server.Start(s.cfg.HTTP.Address)
}

// Shutdown stops the server gracefully.
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Service) initConfig(configFileLocation string) error {
	cfg, err := config.GetConfiguration(configFileLocation)
	if err != nil {
		return fmt.Errorf("failed to load configuration file: %s", err)
	}
	s.cfg = cfg
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

	connString := s.cfg.Database.ConnectionString()
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

// NewService reates a returns a new instance of Service, which is by
// default ready to be executed.
func NewService(configFileLocation string) (*Service, error) {
	s := &Service{}

	// load configuration file
	if err := s.initConfig(configFileLocation); err != nil {
		return nil, err
	}

	// TODO: get this value from the configuration file
	s.log = logger.NewLogger(-1)

	// connect to database
	if s.cfg.Database.Enabled {
		if err := s.initDatabase(); err != nil {
			return nil, err
		}
	}

	// web server config
	s.initWebServer()

	return s, nil
}
