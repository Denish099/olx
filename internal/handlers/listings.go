package handlers

import (
	"database/sql"
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

	// FIX: the json tag was "desciption" — clients were literally receiving
	// a misspelled field name. Struct tags are just strings to the compiler,
	// so nothing warns you. Read them out loud when you write them.
	Description string `json:"description"`

	// FIX: this was `Price string` while the column is BIGINT.
	// It never crashed, which is the scary part: database/sql silently
	// converts int64 -> string when you scan into a *string. So the API
	// happily shipped `"price": "1500"` (a JSON *string*) and every client
	// had to parse it before doing arithmetic. Silent conversions are worse
	// than errors — no stack trace, just quietly wrong data.
	Price int64 `json:"price"`

	City string `json:"city"`

	// STYLE: Go identifiers are MixedCaps, never snake_case.
	// Created_at -> CreatedAt. The json tag keeps the wire format as
	// created_at, so nothing changes for whoever consumes this API —
	// that separation between Go name and JSON name is exactly what
	// struct tags are for.
	CreatedAt time.Time `json:"created_at"`
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

// STYLE: pointer receivers (*ListingHandler) on both methods now.
// Before, the constructor returned *ListingHandler but the methods used a
// value receiver. Go auto-dereferences so it compiled, but the convention
// is one or the other per type. Mixing is where "I set the field and it
// didn't stick" bugs come from — a value receiver gets a COPY of the
// struct, so any mutation is thrown away when the method returns.
func (lh *ListingHandler) List(w http.ResponseWriter, r *http.Request) {
	// r.Context() is the request-scoped context: it is cancelled the moment
	// the client hangs up. Passing it to QueryContext means Postgres gets
	// told to stop work nobody is waiting for any more.
	ctx := r.Context()
	requestID := middleware.RequestIDFromContext(ctx)

	rows, err := lh.db.QueryContext(ctx,
		`SELECT id, title, description, price, city, created_at
		   FROM listings
		  ORDER BY created_at DESC
		  LIMIT 50`)
	if err != nil {
		// Log the real error (for you), return a vague one (for them).
		lh.logger.Error("listings.list: query failed", "requestId", requestID, "err", err)
		httpx.Error(w, httpx.CodeInternalError, "something went wrong")
		return
	}
	defer rows.Close()

	// Deliberately `[]listing{}` and not `var listings []listing`.
	// A nil slice marshals to JSON `null`; an empty slice marshals to `[]`.
	// Clients doing `data.map(...)` break on null. You had this right —
	// keeping the note because it's a genuinely easy one to "tidy up" wrong.
	listings := []listing{}

	for rows.Next() {
		var l listing
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.Price, &l.City, &l.CreatedAt); err != nil {
			lh.logger.Error("listings.list: scan failed", "scanned", len(listings), "requestId", requestID, "err", err)
			// FIX: this branch (and the one below) still used http.Error, so
			// your API returned plain text here and JSON elsewhere. A client
			// can't parse both. That inconsistency is the entire reason the
			// httpx package exists — so use it everywhere, no exceptions.
			httpx.Error(w, httpx.CodeInternalError, "something went wrong")
			return
		}
		listings = append(listings, l)
	}

	// rows.Err() catches an error that ENDED the loop early — a dropped
	// connection mid-stream, say. rows.Next() just returns false in that
	// case, so without this check you'd return 200 with a partial list and
	// never find out. Good that you already had it.
	if err := rows.Err(); err != nil {
		lh.logger.Error("listings.list: rows iteration failed", "requestId", requestID, "err", err)
		httpx.Error(w, httpx.CodeInternalError, "something went wrong")
		return
	}

	// FIX: was `w.Header().Set("Content-Type", "appication/json")` — missing
	// the 'l', so nothing treated the body as JSON. httpx.JSON sets it
	// correctly, and now that typo can only ever exist in one file.
	httpx.JSON(w, http.StatusOK, listings)
}

func (lh *ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.RequestIDFromContext(ctx)
	id := r.PathValue("id")

	// FIX #1 — no input validation.
	// `DELETE /listings/abc` sent "abc" straight to Postgres, which rejected
	// it with `invalid input syntax for type uuid`. You then reported that
	// as 500 "internal server error" — but nothing internal went wrong. The
	// CLIENT sent garbage. 5xx means "my fault", 4xx means "your fault", and
	// mixing them up will wreck your alerting later (you'll page yourself at
	// 3am because someone fuzzed your URL).
	//
	// Rule: validate at the edge, so a bad request never reaches the DB.
	if _, err := uuid.Parse(id); err != nil {
		httpx.Error(w, httpx.CodeInvalidID, "id must be a valid uuid")
		return
	}

	res, err := lh.db.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		lh.logger.Error("listings.delete: exec failed", "listingId", id, "requestId", requestID, "err", err)
		httpx.Error(w, httpx.CodeInternalError, "something went wrong")
		return
	}

	// FIX #2 — the sql.Result was discarded with `_`.
	// A DELETE that matches zero rows is NOT an error in SQL. So deleting an
	// id that never existed returned 204 "yep, deleted it" — a lie, and one
	// that hides bugs in whatever is calling you. RowsAffected is how you
	// find out whether anything actually happened.
	n, err := res.RowsAffected()
	if err != nil {
		lh.logger.Error("listings.delete: rowsAffected failed", "listingId", id, "requestId", requestID, "err", err)
		httpx.Error(w, httpx.CodeInternalError, "something went wrong")
		return
	}
	if n == 0 {
		httpx.Error(w, httpx.CodeNotFound, "listing not found")
		return
	}

	lh.logger.Info("listings.delete: deleted", "listingId", id, "requestId", requestID)

	// 204 No Content means success with deliberately no body.
	// Don't write one after this — net/http will discard it and log a
	// "superfluous WriteHeader" style complaint.
	w.WriteHeader(http.StatusNoContent)
}
