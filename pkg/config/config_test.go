package config_test

// config_test.go — контрактные тесты для pkg/config.
//
// Контракт:
//  1. Load читает env-переменные в struct по тегу `env:"KEY"`.
//  2. Load применяет значения по умолчанию (`env-default:"value"`).
//  3. Load возвращает ошибку если обязательная переменная отсутствует (`env-required:"true"`).
//  4. Load читает переменные из .env файла если передан путь.

import (
	"os"
	"testing"

	"github.com/miilkaa/portsmith/pkg/config"
)

type testConfig struct {
	Host     string `env:"TEST_HOST"     env-default:"localhost"`
	Port     int    `env:"TEST_PORT"     env-default:"8080"`
	Debug    bool   `env:"TEST_DEBUG"    env-default:"false"`
	Required string `env:"TEST_REQUIRED" env-required:"true"`
}

func TestLoad_readsEnvVars(t *testing.T) {
	t.Setenv("TEST_HOST", "example.com")
	t.Setenv("TEST_PORT", "9090")
	t.Setenv("TEST_REQUIRED", "present")

	var cfg testConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoad_appliesDefaults(t *testing.T) {
	// Clear any leftover variables.
	os.Unsetenv("TEST_HOST")
	os.Unsetenv("TEST_PORT")
	t.Setenv("TEST_REQUIRED", "present")

	var cfg testConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected default host localhost, got %s", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.Debug != false {
		t.Error("expected debug false by default")
	}
}

func TestLoad_errorOnMissingRequired(t *testing.T) {
	os.Unsetenv("TEST_REQUIRED")

	var cfg testConfig
	err := config.Load(&cfg)
	if err == nil {
		t.Error("expected error for missing required variable, got nil")
	}
}
