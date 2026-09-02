package main

import (
	"log"
	"os"

	"github.com/Denish099/olx/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatal("usage : make migrate <up/down>")
	}

	cfg := config.MustLoad()

	m, err := migrate.New(
		"file://migrations",
		cfg.DATABASE_URL)

	if err != nil {
		log.Fatalf("migration.new: %v", err)
	}
	// fmt.Println(os.Args)

	switch os.Args[1] {
	case "up":
		if err := m.Up(); err != nil {
			log.Fatalf("%v", err)
		}
	case "down":
		// in this down rollback all the way down so all data would be lost. so use steps
		if err := m.Steps(-1); err != nil {
			log.Fatalf("%v", err)
		}
	default:
		log.Fatalf("invalid command : %s", os.Args[1])
	}

	// fmt.Println("migration is running")
}
