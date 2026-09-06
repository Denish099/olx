package handlers

import "net/http"

func Health(w http.ResponseWriter, r *http.Request) {
	// FIX: was "appication/json" (missing the l), so clients did not parse it as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"status": "all ok"
	}`))
}
