package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	DefaultAppName          = "MyApp"
	defaultHttpAddress      = ":8080"
	DefaultJwtExpiration    = time.Hour * 48
	DefaultJwtSigningKey    = "default-signing-key"
	DefaultHttpReadTimeout  = time.Second * 5
	DefaultHttpWriteTimeout = time.Second * 10
	DefaultHttpIdleTimeout  = time.Minute * 2
)

type (
	// ConfigTemplate defines the interface for custom Configuration structures.
	ConfigTemplate interface {
		GetApp() *AppConfig
		GetHTTP() *HttpConfig
		GetDatabase() *DatabaseConfig
		GetCustomValue(name string) (interface{}, bool)
		SetDefaults()
	}

	// Config contains the global service settings.
	Config struct {
		App      AppConfig
		HTTP     HttpConfig
		Database DatabaseConfig
		Custom   map[string]interface{}
	}

	// AppConfig holds the application settings
	AppConfig struct {
		Name          string
		IsDevelopment bool
		LogLevel      int
		LogFormat     string
	}

	// HttpConfig holds the HTTP server related configuration
	HttpConfig struct {
		Address            string
		ReadTimeout        time.Duration
		WriteTimeout       time.Duration
		IdleTimeout        time.Duration
		CustomHealthCheck  bool
		JwtEnabled         bool
		JwtSigningKey      string
		JwtTokenExpiration time.Duration
	}

	// DatabaseConfig stores the database configuration
	DatabaseConfig struct {
		Enabled  bool
		Driver   string
		Address  string
		Port     uint16
		User     string
		Password string
		Database string
	}
)

func (cfg *Config) GetApp() *AppConfig {
	return &cfg.App
}

func (cfg *Config) GetHTTP() *HttpConfig {
	return &cfg.HTTP
}

func (cfg *Config) GetDatabase() *DatabaseConfig {
	return &cfg.Database
}

func (cfg *Config) GetCustom() map[string]interface{} {
	return cfg.Custom
}

// GetCustomValue returns the value of a custom configuration, if it's
// present. The return type will depend of how the value is stored in the yaml.
func (cfg *Config) GetCustomValue(name string) (interface{}, bool) {
	val, ok := cfg.Custom[name]
	return val, ok
}

// ConnectionString returns the connection string based on the configured driver.
// Currently supported drivers are:
//   - mysql
func (dbCfg *DatabaseConfig) ConnectionString() string {
	if dbCfg.Enabled {
		switch strings.ToLower(dbCfg.Driver) {

		case "mysql":
			return fmt.Sprintf(
				"%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True",
				dbCfg.User,
				dbCfg.Password,
				dbCfg.Address,
				dbCfg.Database,
			)
		}
	}
	return ""
}

// SetDefaults checks the configuration values and sets some default where needed.
func (cfg *Config) SetDefaults() {
	if app := cfg.GetApp(); app != nil {
		if app.Name == "" {
			app.Name = DefaultAppName
		}
		if app.LogFormat == "" {
			app.LogFormat = "console"
		}
	}

	if httpCfg := cfg.GetHTTP(); httpCfg != nil {
		if httpCfg.Address == "" {
			httpCfg.Address = defaultHttpAddress
		}
		if httpCfg.JwtEnabled {
			if httpCfg.JwtSigningKey == "" {
				httpCfg.JwtSigningKey = DefaultJwtSigningKey // DON'T USE THIS ON PRODUCTION!
			}
			if httpCfg.JwtTokenExpiration == 0 {
				httpCfg.JwtTokenExpiration = DefaultJwtExpiration
			}
		}
		if httpCfg.ReadTimeout == 0 {
			httpCfg.ReadTimeout = DefaultHttpReadTimeout
		}
		if httpCfg.WriteTimeout == 0 {
			httpCfg.WriteTimeout = DefaultHttpWriteTimeout
		}
		if httpCfg.IdleTimeout == 0 {
			httpCfg.IdleTimeout = DefaultHttpIdleTimeout
		}
	}
}

// GetConfiguration loads the service configuration.
func GetConfiguration(configFilePath string, cfgTemplate ConfigTemplate) error {
	// parse config file name
	dir, fileName, fileExt := parseFileName(configFilePath)
	if fileName != "" {
		viper.SetConfigName(fileName)
	} else {
		viper.SetConfigName("config") // use 'config' as default
	}
	if fileExt != "" {
		viper.SetConfigType(fileExt)
	} else {
		viper.SetConfigType("yaml") // use 'yaml' as default
	}

	viper.AddConfigPath(dir)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(cfgTemplate); err != nil {
		return err
	}

	return nil
}

func parseFileName(f string) (string, string, string) {
	dir := filepath.Dir(f)
	file := filepath.Base(f)
	ext := filepath.Ext(file)

	if ext != "" {
		sepIndex := strings.LastIndex(file, ".")
		fileName := file[0:sepIndex]
		return dir, fileName, strings.Replace(ext, ".", "", 1)
	}

	return dir, file, ""
}
