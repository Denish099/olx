package handlers

import "time"

type CreateListingsReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       uint64 `json:"price"`
	City        string `json:"city"`
}

type CreateListingsRes struct {
	ID         string    `json:"id"`
	Created_at time.Time `json:"created_at"`
}
