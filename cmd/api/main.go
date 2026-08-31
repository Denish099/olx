package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Denish099/olx/internal/handlers"
)

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.Health)

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
