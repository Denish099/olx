package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Denish099/olx/internal/config"
	"github.com/Denish099/olx/internal/db"
	"github.com/Denish099/olx/internal/handlers"
	"github.com/Denish099/olx/internal/middleware"
)

// The `func main` + `func run() error` split is a very common Go pattern,
// and the reason is subtle but important:
//
// log.Fatalf and os.Exit terminate the process IMMEDIATELY — deferred
// functions do NOT run. So the old code's cleanup would have been silently
// skipped on any error path. By keeping all the work (and all the defers)
// inside run(), the defers fire on the way out, and main() is the single
// place allowed to exit.
func main() {
	if err := run(); err != nil {
		log.Fatalf("api: %v", err)
	}
}

func run() error {
	// FIX: the logger is built FIRST now.
	// Previously it was created *after* the DB connect, so a connection
	// failure fell through to log.Fatalf — a line of plain text in the
	// middle of an otherwise all-JSON log stream. Any log aggregator
	// (Loki, CloudWatch, Datadog) chokes on that exact line, which is the
	// one line you most need when the app won't boot.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))

	cfg := config.MustLoad()

	// FIX: this used to be `db, err := db.Connect(...)`.
	// That SHADOWS the imported `db` package with a local variable — legal
	// Go, compiles fine, but the package is now unreachable for the rest of
	// the function. Name a variable after what it holds, not after the
	// package it came from.
	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		// %w wraps the error so callers can still errors.Is / errors.As it.
		// Use %w when wrapping, %v when you're just printing.
		return fmt.Errorf("db.connect: %w", err)
	}
	// FIX: was missing entirely. It mattered less when the process died
	// immediately after any error — but now that we shut down gracefully,
	// there's a real window where the pool should be closed after the
	// server has finished draining. defer runs LIFO, so this fires last.
	defer pool.Close()

	logger.Info("database connected")

	mux := http.NewServeMux()
	lh := handlers.NewListingHandler(pool, logger)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /listings", lh.List)
	mux.HandleFunc("DELETE /listings/{id}", lh.Delete)

	handler := middleware.RequestID(mux)

	// FIX: &http.Server{...} rather than http.Server{...}.
	// http.Server contains a sync.Mutex and internal state, so copying one
	// is a latent bug — `go vet` flags copied locks. Yours worked because a
	// local variable is addressable, but always take the pointer here.
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- graceful shutdown -------------------------------------------
	// Before this, Ctrl+C killed the process instantly and any request
	// mid-flight was cut off — the client sees a dropped connection, and a
	// half-finished DB write is anyone's guess. This is the standard Go
	// shutdown pattern; learn it once and paste it into every service.

	// signal.NotifyContext hands you a context that is cancelled when one
	// of these signals arrives. SIGTERM is what Docker and Kubernetes send
	// to stop a container, so handling it is precisely what makes a rolling
	// deploy not drop requests. os.Interrupt is your Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ListenAndServe blocks forever, so it goes in its own goroutine and
	// reports back over a channel.
	// The channel is BUFFERED (size 1) on purpose: if we exit via the
	// shutdown path below, nobody is left reading this channel, and an
	// unbuffered send would block that goroutine forever — a leak.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		serverErr <- srv.ListenAndServe()
	}()

	// select waits on whichever happens first: the server dying on its own,
	// or a shutdown signal arriving.
	select {
	case err := <-serverErr:
		// FIX: ListenAndServe returns http.ErrServerClosed on a CLEAN stop.
		// The old `if err != nil { log.Fatalf }` would therefore report a
		// perfectly normal shutdown as a crash.
		//
		// errors.Is is how you check for a specific sentinel error in Go.
		// Never compare error strings (err.Error() == "..."), and don't use
		// == either once errors get wrapped with %w — errors.Is unwraps.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}

	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")

		// A FRESH context, because ctx is already cancelled at this point.
		// This 10s is the deadline for in-flight requests to finish; after
		// that Shutdown gives up and returns an error.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown timed out, forcing close", "err", err)
			_ = srv.Close() // hard stop: kill remaining connections
		}
		logger.Info("server stopped cleanly")
	}

	return nil
}
