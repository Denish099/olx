package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type listing struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"desciption"`
	Price       string    `json:"price"`
	City        string    `json:"city"`
	Created_at  time.Time `json:"created_at"`
}

func List(db *sql.DB) http.HandlerFunc {
	// the List is th handler function for main listing hanlder due to dependency injection
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(
			`SELECT id,title,description,price,city,created_at 
			FROM listings
			ORDER BY created_at DESC
			LIMIT 50`)

		if err != nil {
			log.Printf("query: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		listings := []listing{}
		for rows.Next() {
			var l listing
			if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.Price, &l.City, &l.Created_at); err != nil {
				log.Printf("rows.scan: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			listings = append(listings, l)
		}

		if err := rows.Err(); err != nil {
			log.Printf("rows.err: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "appication/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(listings)

	}
}
