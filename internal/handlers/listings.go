package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Denish099/olx/internal/httpx"
	"github.com/Denish099/olx/internal/middleware"
)

type listing struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// FIX: tag was "desciption" (typo) - clients got a misspelled field.
	Description string `json:"description"`
	// FIX: was `string`, but the column is BIGINT. It didn't crash because
	// database/sql quietly converts int64 -> string, but the API sent
	// "price": "1500" (a JSON string) instead of 1500.
	Price      string    `json:"price"`
	City       string    `json:"city"`
	Created_at time.Time `json:"created_at"`
}

type ListingHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewListingHandler(db *sql.DB, logger *slog.Logger) *ListingHandler {
	return &ListingHandler{
		db:     db,
		logger: logger,
	}
}

func (lh ListingHandler) /*method reciever */ List(w http.ResponseWriter, r *http.Request) {
	// request scope context
	ctx := r.Context()
	rows, err := lh.db.QueryContext(ctx,
		`SELECT id,title,description,price,city,created_at
			FROM listings
			ORDER BY created_at DESC
			LIMIT 50`)

	if err != nil {
		lh.logger.Error("query.error", "err", err)

		httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
		return
	}
	defer rows.Close()

	listings := []listing{}
	for rows.Next() {
		var l listing
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.Price, &l.City, &l.Created_at); err != nil {
			lh.logger.Error("rows.scan failed", "listings", len(listings), "err", err)

			// FIX: was http.Error, so this path returned plain text while the
			// one above returned JSON. Clients can't parse both.
			httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
			return
		}

		listings = append(listings, l)
	}

	if err := rows.Err(); err != nil {
		lh.logger.Error("rows.error: ", "err", err)

		httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
		return
	}
	// FIX: was "appication/json" (missing the l), so nothing parsed it as JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listings)

}
func (lh ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestId := middleware.RequestIdFromContext(ctx)
	id := r.PathValue("id")

	// FIX: no validation before. DELETE /listings/abc sent "abc" to Postgres,
	// which errored with "invalid input syntax for type uuid", and you
	// reported that as 500. But nothing broke on your side - the client sent
	// garbage, so it's a 400.
	if _, err := uuid.Parse(id); err != nil {
		httpx.Error(w, http.StatusBadRequest, "id must be a valid uuid", httpx.CodeInvalidId)
		return
	}

	res, err := lh.db.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)

	if err != nil {
		lh.logger.Error("delete fail", "listing id", id, "requestId", requestId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
		return
	}

	// FIX: the sql.Result was thrown away with `_`. A DELETE matching zero
	// rows is not an error in SQL, so deleting an id that never existed
	// returned 204 "deleted it" - a lie. RowsAffected tells you the truth.
	n, err := res.RowsAffected()
	if err != nil {
		lh.logger.Error("rowsAffected fail", "listing id", id, "requestId", requestId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
		return
	}
	if n == 0 {
		httpx.Error(w, http.StatusNotFound, "listing not found", httpx.CodeNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
