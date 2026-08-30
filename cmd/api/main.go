package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Denish099/olx/internal/config"
)

func main() {

	cfg := config.MustLoad()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "appication/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "all ok"
		}`))
	})

	srv := http.Server{
		Addr:         ":" + cfg.PORT,
		Handler:      mux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 60,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
