// Package config provides environment-based configuration loading.
//
// Define a config struct with env tags:
//
//	type Config struct {
//	    Port        int    `env:"PORT"         env-default:"8080"`
//	    DatabaseURL string `env:"DATABASE_URL" env-required:"true"`
//	    LogLevel    string `env:"LOG_LEVEL"    env-default:"info"`
//	}
//
// Load from environment:
//
//	var cfg Config
//	if err := config.Load(&cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Load from a .env file (variables in environment take precedence):
//
//	if err := config.LoadFile(".env", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Supported tag options:
//   - env:"VAR_NAME" — environment variable name
//   - env-default:"value" — default value when variable is not set
//   - env-required:"true" — return error when variable is not set
package config
