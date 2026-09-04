package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Denish099/olx/internal/config"
	"github.com/Denish099/olx/internal/db"
	"github.com/Denish099/olx/internal/handlers"
	"github.com/Denish099/olx/internal/middleware"
)

func main() {
	cfg := config.MustLoad()

	db, err := db.Connect(cfg.DATABASE_URL)

	// we should never do database migrations here because there could be many instance of this running each calling databaseMigration.
	if err != nil {
		log.Fatalf("main.db.connect: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))

	fmt.Println("database connected")

	mux := http.NewServeMux()

	lh := handlers.NewListingHandler(db, logger)

	//constructor pattern added dependency only once . not again and again
	mux.HandleFunc("GET /healthz", handlers.Health)

	mux.HandleFunc("GET /listings", lh.List)

	mux.HandleFunc("DELETE /listings/{id}", lh.Delete)

	handler := middleware.RequestId(mux)

	srv := http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 60,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
