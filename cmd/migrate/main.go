package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Denish099/olx/internal/config"
	"github.com/golang-migrate/migrate/v4"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: go run ./cmd/migrate <up|down>")
	}

	cfg := config.MustLoad()

	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("migrate.new: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate.close source: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate.close database: %v", dbErr)
		}
	}()

	cmd := os.Args[1]

	switch cmd {
	case "up":

		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("up: %w", err)
		}

	case "down":

		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("down: %w", err)
		}

	default:

		return fmt.Errorf("invalid command %q (want up or down)", cmd)
	}

	log.Printf("migrate %s: ok", cmd)
	return nil
}
