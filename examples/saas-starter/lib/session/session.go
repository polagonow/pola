// Package session reads and writes the authenticated user id in the JWT session
// cookie. Both bridged actions and REST routes use it, so the session contract
// lives in one place.
package session

import (
	"context"

	"github.com/polagonow/pola/middleware/jwt"
)

// Set signs the user id into the JWT session cookie.
func Set(ctx context.Context, userID uint) {
	jwt.Set(ctx, map[string]any{"userId": userID})
}

// Clear removes the session cookie (sign-out).
func Clear(ctx context.Context) { jwt.Clear(ctx) }

// UserID returns the authenticated user id from the JWT session, if any.
func UserID(ctx context.Context) (uint, bool) {
	v, ok := jwt.Get(ctx)["userId"]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return uint(n), true
	case int:
		return uint(n), true
	case int64:
		return uint(n), true
	case uint:
		return n, true
	}
	return 0, false
}
