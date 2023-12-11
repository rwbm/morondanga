package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type (
	// Config contains the global service settings.
	Config struct {
		App      AppConfig
		HTTP     HttpConfig
		Database DatabaseConfig
		Custom   map[string]interface{}
	}

	// AppConfig holds the application settings
	AppConfig struct {
		Name  string
		Debug bool
	}

	// HttpConfig holds the HTTP server related configuration
	HttpConfig struct {
		Address           string
		ReadTimeout       time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
		CustomHealthCheck bool
		JwtEnabled        bool
		JwtSigningKey     string
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

// GetCustom returns the value of a custom configuration, if it's
// present. The return type will depend of how the value is stored in the yaml.
func (cfg *Config) GetCustom(name string) (interface{}, bool) {
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

// GetConfiguration loads the service configuration.
func GetConfiguration(configFileLocation string) (*Config, error) {
	var c Config

	// Load the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configFileLocation != "" {
		viper.AddConfigPath(configFileLocation)
	}
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	viper.AddConfigPath("../config")

	// Load env variables
	// viper.SetEnvPrefix(prefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return &c, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return &c, err
	}

	return &c, nil
}
