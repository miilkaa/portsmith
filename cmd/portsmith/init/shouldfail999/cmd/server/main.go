package main

import (
	"log"

	"github.com/miilkaa/portsmith/pkg/config"
	"github.com/miilkaa/portsmith/pkg/database"
	"github.com/miilkaa/portsmith/pkg/server"

	// Uncomment after creating your first package:
	// "shouldfail999/internal/<package>"
)

// Config holds all application settings loaded from environment variables.
type Config struct {
	Port        int    `env:"PORT"         env-default:"8080"`
	DatabaseURL string `env:"DATABASE_URL" env-required:"true"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(database.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	// Register domain models for AutoMigrate:
	// database.Register(db, &orders.Order{})

	srv := server.New(server.Config{Port: cfg.Port})

	// Wire your handlers here:
	// repo := orders.NewRepository(db.DB())
	// svc  := orders.NewService(repo)
	// h    := orders.NewHandler(svc)
	// h.Routes(srv.Router().Group("/api/v1"))

	_ = db // remove after wiring
	log.Fatal(srv.Run())
}
