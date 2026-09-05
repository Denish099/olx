package handlers

import (
	"net/http"

	"github.com/Denish099/olx/internal/httpx"
)

// Health is a liveness probe — it answers "is this process up and serving?".
//
// Note it deliberately does NOT ping the database. A liveness check that
// fails when a dependency is down will get your container restart-looped by
// Kubernetes for something a restart can't fix. If you want a DB check, add
// a SEPARATE /readyz that reports "ready to take traffic".
func Health(w http.ResponseWriter, r *http.Request) {
	// FIX: the Content-Type here was "appication/json" (missing the 'l'),
	// so clients wouldn't auto-parse the body. Going through httpx.JSON
	// means the header string exists in exactly one place in the codebase.
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
