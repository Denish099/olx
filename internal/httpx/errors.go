package httpx

import (
	"encoding/json"
	"net/http"
)

type Code string

// All the codes from your table. Comment on the right is the status you
// pass to Error() with each one.
const (
	CodeInvalidId        Code = "invalid_id"        // 400
	CodeMalformedJson    Code = "malformed_json"    // 400
	CodeUnauthenticated  Code = "unauthenticated"   // 401
	CodeForbidden        Code = "forbidden"         // 403
	CodeNotFound         Code = "not_found"         // 404
	CodeConflict         Code = "conflict"          // 409
	CodeValidationFailed Code = "validation_failed" // 422
	CodeRateLimited      Code = "rate_limited"      // 429
	CodeInternalError    Code = "internal_error"    // 500
)

type errorEnvolpe struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, status int, message string, code Code) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(errorEnvolpe{
		errorPayload{
			Code:    code,
			Message: message,
		},
	})
}
