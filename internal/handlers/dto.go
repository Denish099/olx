package handlers

import (
	"fmt"
	"strings"
	"time"
)

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

type validationError struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}

func (e *validationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

/* we are returning validateError struct instead of error because we implement error interface in validateError struct*/

func (req CreateListingsReq) validate() error {
	if strings.TrimSpace(req.Title) == "" {
		return &validationError{
			Field: "title",
			Msg:   "must not be empty",
		}
	}

	return nil
}
