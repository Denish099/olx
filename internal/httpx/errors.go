// Package httpx holds small helpers for writing HTTP responses.
//
// Why a package at all? Because "how an error looks on the wire" is a
// decision your API makes ONCE. If every handler hand-rolls its own
// response, half of them end up plain text and half JSON, and your clients
// can't parse either reliably. One helper = one shape, forever.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Code is a machine-readable error code.
//
// Note it is `type Code string`, not just `string`. That's a defined type,
// and Go gives you compile-time safety for free: a function taking a Code
// will REFUSE a bare string literal. So you physically cannot typo
// "not_fond" into a response — you have to use one of the constants below.
type Code string

// The full catalogue of error codes this API can return.
//
// Two audiences, two identifiers, and they are not the same thing:
//   - the HTTP status is for machines in the middle — proxies, CDNs,
//     retry logic, your monitoring dashboards
//   - the Code is for YOUR client — so a frontend can switch on
//     "validation_failed" without string-matching English messages that
//     you will inevitably reword later
//
// (Renamed CodeInvalidId -> CodeInvalidID: Go capitalises initialisms
// whole. It's URL, ID, HTTP, API — never Url, Id, Http. This is one of the
// few naming rules the community is actually strict about.)
const (
	CodeInvalidID        Code = "invalid_id"        // 400
	CodeMalformedJSON    Code = "malformed_json"    // 400
	CodeUnauthenticated  Code = "unauthenticated"   // 401
	CodeForbidden        Code = "forbidden"         // 403
	CodeNotFound         Code = "not_found"         // 404
	CodeConflict         Code = "conflict"          // 409
	CodeValidationFailed Code = "validation_failed" // 422
	CodeRateLimited      Code = "rate_limited"      // 429
	CodeInternalError    Code = "internal_error"    // 500
)

// statusFor pairs each code with its HTTP status.
//
// Keeping the pairing in ONE table is the whole trick. If every call site
// had to remember "422 goes with validation_failed", sooner or later one
// handler sends 400 and another sends 422 for the same situation, and now
// your client has to handle both.
var statusFor = map[Code]int{
	CodeInvalidID:        http.StatusBadRequest,          // 400
	CodeMalformedJSON:    http.StatusBadRequest,          // 400
	CodeUnauthenticated:  http.StatusUnauthorized,        // 401
	CodeForbidden:        http.StatusForbidden,           // 403
	CodeNotFound:         http.StatusNotFound,            // 404
	CodeConflict:         http.StatusConflict,            // 409
	CodeValidationFailed: http.StatusUnprocessableEntity, // 422
	CodeRateLimited:      http.StatusTooManyRequests,     // 429
	CodeInternalError:    http.StatusInternalServerError, // 500
}

// StatusFor returns the HTTP status that belongs with a code.
func StatusFor(code Code) int {
	// This is the "comma ok" form of a map read, and you want it here.
	// A plain `statusFor[code]` returns 0 for a missing key — and 0 is a
	// perfectly valid int, so you'd have no way to tell "not in the map"
	// from "in the map with value 0". Same idea as comma-ok on type
	// assertions and channel receives; it shows up everywhere in Go.
	if s, ok := statusFor[code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// errorEnvelope wraps the payload so the body is
//
//	{"error": {"code": "not_found", "message": "listing not found"}}
//
// rather than a bare {"code": ..., "message": ...}. Nesting under a key
// leaves room to add siblings later ("meta", "trace_id") without breaking
// clients that are already parsing .error.code.
//
// (Was spelled errorEnvolpe. Unexported types get typo'd names too, and
// those are the names you retype for the rest of the project's life.)
type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

// Error writes a JSON error response.
//
// The signature changed: it used to be Error(w, status, message, code),
// which let a caller pair 404 with "internal_error" by accident. Now the
// status is DERIVED from the code, so that mismatch is impossible. When a
// value can be computed from another, don't ask the caller for both.
//
// `message` stays free-form because it's the human-facing half — but keep
// it vague for 5xx. Never leak driver errors or SQL to a client; log those.
func Error(w http.ResponseWriter, code Code, message string) {
	write(w, StatusFor(code), errorEnvelope{
		Error: errorPayload{
			Code:    code,
			Message: message,
		},
	})
}

// JSON writes any value as a successful JSON response.
func JSON(w http.ResponseWriter, status int, v any) {
	write(w, status, v)
}

// write is the single place this package touches the ResponseWriter.
func write(w http.ResponseWriter, status int, v any) {
	// IMPORTANT: marshal FIRST, then write.
	//
	// The old handler code did:
	//     w.WriteHeader(http.StatusOK)
	//     json.NewEncoder(w).Encode(listings)
	// WriteHeader flushes the status line to the client immediately. So if
	// Encode then fails halfway, you have already promised "200 OK" and can
	// only truncate the body — the client gets broken JSON and a success
	// status. Buffering means a marshal failure can still become a clean 500.
	buf, err := json.Marshal(v)
	if err != nil {
		// Hand-written fallback rather than calling Error(), because Error()
		// calls write() and we'd have built ourselves an infinite loop.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"something went wrong"}}`))
		return
	}

	// http.Header canonicalises keys for you, so "content-type" and
	// "Content-Type" are the same key — your lowercase version worked fine.
	// Set (not Add) so a second call replaces rather than sends it twice.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}
