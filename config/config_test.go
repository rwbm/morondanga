package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Config{}
	cfgFilePath := "testdata/config.yml"

	err := GetConfiguration(cfgFilePath, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "MyApp", cfg.GetApp().Name)
	assert.Equal(t, "127.0.0.1:8080", cfg.GetHTTP().Address)

	// get a custom entry value
	myKey, ok := cfg.GetCustomValue("mykey")
	assert.Equal(t, true, ok)
	assert.Equal(t, "key value", myKey)
}

func TestCustomConfig(t *testing.T) {
	type myCustomConfiguration struct {
		Config
		MyCustomSection struct {
			Key1 string
			Key2 int
		}
	}

	cfg := myCustomConfiguration{}
	cfgFilePath := "testdata/config-custom.yml"

	err := GetConfiguration(cfgFilePath, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "MyApp", cfg.GetApp().Name)
	assert.Equal(t, "value", cfg.MyCustomSection.Key1)
	assert.Equal(t, 12345, cfg.MyCustomSection.Key2)
}

func TestEnvironmentOverride(t *testing.T) {
	cfg := Config{}
	cfgFilePath := "testdata/config.yml"

	// set env var to override the config value
	t.Setenv("HTTP_JWTSIGNINGKEY", "new-key-value")

	err := GetConfiguration(cfgFilePath, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "new-key-value", cfg.GetHTTP().JwtSigningKey)
}

func TestParseFileName(t *testing.T) {
	test := []struct {
		input string
		want  []string
	}{
		{input: "/User/test/morondanga/config.sample.yml", want: []string{"/User/test/morondanga", "config.sample", "yml"}},
		{input: "/User/test/morondanga/config.yml", want: []string{"/User/test/morondanga", "config", "yml"}},
		{input: "/User/test/morondanga/configyml", want: []string{"/User/test/morondanga", "configyml", ""}},
		{input: "/User/test/morondanga/mysettings.xml", want: []string{"/User/test/morondanga", "mysettings", "xml"}},
		{input: "onlyfilename.json", want: []string{".", "onlyfilename", "json"}},
	}
	for _, cases := range test {
		t.Run(cases.input, func(t *testing.T) {
			dir, name, ext := parseFileName(cases.input)
			assert.Equal(t, cases.want[0], dir)
			assert.Equal(t, cases.want[1], name)
			assert.Equal(t, cases.want[2], ext)
		})
	}
}

func TestDatabaseConnectionStringPostgres(t *testing.T) {
	cfg := DatabaseConfig{
		Enabled:  true,
		Driver:   "postgres",
		Address:  "localhost:5432",
		User:     "postgres",
		Password: "postgres",
		Database: "food_bot",
	}

	dsn := cfg.ConnectionString()
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/food_bot?sslmode=disable", dsn)
}
