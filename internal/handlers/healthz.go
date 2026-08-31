package handlers

import "net/http"

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "appication/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"status": "all ok"
	}`))
}
