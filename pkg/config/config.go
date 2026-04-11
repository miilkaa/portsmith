// Package config provides a thin wrapper around cleanenv for loading
// application configuration from environment variables.
//
// Define a config struct with env tags and call Load:
//
//	type Config struct {
//	    Port     int    `env:"PORT"     env-default:"8080"`
//	    DSN      string `env:"DATABASE_URL" env-required:"true"`
//	    LogLevel string `env:"LOG_LEVEL"    env-default:"info"`
//	}
//
//	var cfg Config
//	if err := config.Load(&cfg); err != nil {
//	    log.Fatal(err)
//	}
package config

import "github.com/ilyakaznacheev/cleanenv"

// Load reads environment variables into the provided struct pointer.
// Fields are mapped via `env:"VAR_NAME"` tags.
// Use `env-default:"value"` for defaults, `env-required:"true"` for required fields.
func Load(cfg any) error {
	return cleanenv.ReadEnv(cfg)
}

// LoadFile reads a .env file and then env variables into the provided struct.
// Variables set in the environment take precedence over the file.
func LoadFile(path string, cfg any) error {
	return cleanenv.ReadConfig(path, cfg)
}
