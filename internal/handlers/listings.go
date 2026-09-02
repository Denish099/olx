package handlers

import (
	"database/sql"
	"net/http"
)

func List(db *sql.DB) http.HandlerFunc {
	// the List is th handler function for main listing hanlder due to dependency injection
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all ok"))
	}
}
