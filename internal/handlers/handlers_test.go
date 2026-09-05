package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Your first Go tests. Run them with `go test ./...`.
//
// Nothing to install: httptest is in the standard library. httptest.NewRequest
// builds a fake *http.Request and httptest.NewRecorder gives you a fake
// ResponseWriter that remembers the status, headers and body — so you can call
// a handler as a plain function and assert on what it wrote. No server, no
// port, no database.

// discardLogger throws log output away so test runs stay readable.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthReturnsJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Health(rec, req)

	if rec.Code != http.StatusOK {
		// t.Fatalf stops this test immediately; t.Errorf marks it failed but
		// keeps going. Use Fatalf when later assertions would be meaningless.
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	// This is the assertion that would have caught "appication/json".
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}

// TestDeleteRejectsNonUUID is the test that would have exploded before the
// fixes, and it's worth understanding why:
//
// It builds a BARE request — no middleware wrapping it — so the context has
// no request id in it. The old RequestIDFromContext did an unchecked type
// assertion on a nil value and panicked right there. The comma-ok version
// returns "" instead. (Verified: putting the old line back makes this test
// panic with "interface conversion: interface {} is nil, not string".)
//
// It also pins down the second fix: a non-uuid id is now 400 invalid_id
// rather than a 500 from Postgres choking on the value.
//
// Note the db is nil. That's deliberate and it's a real assertion in
// disguise: if the handler ever forgot to return early on a bad id, it would
// hit lh.db.ExecContext on a nil pointer and this test would blow up.
func TestDeleteRejectsNonUUID(t *testing.T) {
	lh := NewListingHandler(nil, discardLogger())

	req := httptest.NewRequest(http.MethodDelete, "/listings/not-a-uuid", nil)
	// SetPathValue fakes what the router's {id} wildcard would have set.
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	lh.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Assert on the parsed body, not on the raw string — otherwise the test
	// breaks every time you reword a message or reorder a field.
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%q)", err, rec.Body.String())
	}
	if body.Error.Code != "invalid_id" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "invalid_id")
	}
}
