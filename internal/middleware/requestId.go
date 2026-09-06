package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey int

const (
	requestIdKey ctxKey = iota
)

const (
	requestId = "X-Request-Id"
)

func RequestId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestId)
		if id == "" {
			id = uuid.NewString()
		}

		w.Header().Add(requestId, id)
		ctx := context.WithValue(r.Context(), requestIdKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIdFromContext(ctx context.Context) string {
	// FIX: this was `ctx.Value(requestIdKey).(string)`, which PANICS when the
	// value isn't there - ctx.Value returns nil, and nil.(string) blows up.
	// It only worked because every route goes through the middleware.
	// The comma-ok form can't panic: you get "" and ok == false instead.
	requestId, _ := ctx.Value(requestIdKey).(string)
	return requestId
}
