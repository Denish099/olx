package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Denish099/olx/internal/config"
	"github.com/Denish099/olx/internal/db"
	"github.com/Denish099/olx/internal/handlers"
)

func main() {
	cfg := config.MustLoad()

	db, err := db.Connect(cfg.DATABASE_URL)

	// we should never do database migrations here because there could be many instance of this running each calling databaseMigration.
	if err != nil {
		log.Fatalf("main.db.connect: %v", err)
	}
	fmt.Println("database connected")
	mux := http.NewServeMux()

	lh := handlers.NewListingHandler(db)

	//constructor pattern added dependency only once . not again and again
	mux.HandleFunc("GET /healthz", handlers.Health)

	mux.HandleFunc("GET /listings", lh.List)

	mux.HandleFunc("DELETE /listings/{id}", lh.Delete)

	srv := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 60,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
