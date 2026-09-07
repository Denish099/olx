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

	Description string `json:"description"`

	Price      uint64    `json:"price"`
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listings)

}

func (lh ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestId := middleware.RequestIdFromContext(ctx)
	id := r.PathValue("id")

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

func (lh ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestId := middleware.RequestIdFromContext(ctx)
	var req CreateListingsReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lh.logger.Error("failed to decode ", "request_id", requestId, "err", err)
		httpx.Error(w, http.StatusBadRequest, "invalid body", httpx.CodeMalformedJson)
		return
	}

	row := lh.db.QueryRowContext(ctx, `
	INSERT INTO listings (title, description, price, city) VALUES ($1, $2, $3, $4) RETURNING id,created_at`,
		req.Title, req.Description, req.Price, req.City)

	var res CreateListingsRes
	if err := row.Scan(&res.ID, &res.Created_at); err != nil {
		lh.logger.Error("failed to insert", "request_id", requestId, "err", err)

		httpx.Error(w, http.StatusInternalServerError, "something went wrong", httpx.CodeInternalError)
		return
	}

	lh.logger.Info("listings created", "request id", requestId, "listing_id", res.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}
