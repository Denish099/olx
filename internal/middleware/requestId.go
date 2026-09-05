package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ctxKey is an unexported type used *only* as a context key.
//
// This is the standard Go idiom and it's worth understanding why:
// context.WithValue compares keys by == on both type AND value. If you used
// a plain string key like "requestId", any other package in your binary
// using that same string would silently clobber your value. Because ctxKey
// is unexported, no other package can even construct one. Collision-proof.
type ctxKey int

const (
	requestIDKey ctxKey = iota
)

// headerRequestID is the header we read from and echo back.
// Renamed from `requestId` because a package-level const called requestId
// sitting next to a func called RequestId is genuinely confusing to read.
const headerRequestID = "X-Request-Id"

// RequestID is middleware: it takes an http.Handler and returns a wrapped
// http.Handler. That "decorator" shape is the whole middleware pattern in
// Go — no framework required, just functions wrapping functions.
//
// Renamed from RequestId: Go capitalises initialisms whole (ID, URL, HTTP).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reuse an inbound id if the caller (a gateway, another service)
		// already set one — that's what makes a trace stitch together
		// across services instead of restarting at every hop.
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = uuid.NewString()
		}

		// FIX: this was Header().Add, which APPENDS to the header.
		// Today the response has no X-Request-Id yet so it looked fine, but
		// the moment this middleware runs twice in a chain (easy to do
		// accidentally) the response carries two of them, and clients
		// generally read only the first. Set replaces; Add accumulates.
		w.Header().Set(headerRequestID, id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)

		// r.WithContext returns a *copy* of the request with the new
		// context. You never mutate the *http.Request you were handed —
		// that's a hard convention in net/http.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext pulls the id back out for logging.
func RequestIDFromContext(ctx context.Context) string {
	// FIX: this used to be a bare type assertion:
	//
	//     return ctx.Value(requestIdKey).(string)
	//
	// which PANICS when the value isn't there — ctx.Value returns nil, and
	// asserting nil to string blows up. It only ever worked because every
	// route happened to be wrapped by this middleware. The first unit test
	// that built a plain httptest.NewRequest and called the handler
	// directly would have crashed, and the panic would point *here*, not at
	// the missing middleware, so you'd lose an hour to it.
	//
	// The comma-ok form cannot panic: on failure you get the zero value
	// ("" for string) and ok == false. Use it for EVERY type assertion
	// unless you can prove the dynamic type — and you usually can't.
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
