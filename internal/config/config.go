package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds everything the app needs from the environment.
type Config struct {
	// STYLE: Go identifiers are MixedCaps even when they come from
	// SCREAMING_SNAKE env vars — the env var name and the Go field name are
	// separate things. DATABASE_URL -> DatabaseURL, ENV -> Env.
	//
	// And initialisms stay fully capitalised: URL, ID, HTTP, API.
	// It's DatabaseURL, never DatabaseUrl. This one trips up basically
	// everyone coming from another language.
	Env         string
	DatabaseURL string
}

// MustLoad reads config from the environment and panics if anything
// required is missing.
//
// The "Must" prefix is a Go convention meaning "panics instead of returning
// an error" (see regexp.MustCompile, template.Must). It is only appropriate
// for things that run ONCE at startup, like this — a bad config should stop
// the process immediately rather than fail mysteriously on request #4000.
// Never use a Must* function inside a request handler.
func MustLoad() Config {
	// The error is deliberately ignored: in production there is no .env
	// file at all, the values come from the real environment, and that is
	// not a failure. Assigning to `_` documents that the ignore is on
	// purpose rather than an oversight.
	_ = godotenv.Load()

	env := os.Getenv("ENV")
	if env == "" {
		panic("ENV is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic("DATABASE_URL is required")
	}

	return Config{
		Env:         env,
		DatabaseURL: databaseURL,
	}
}
