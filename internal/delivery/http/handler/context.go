package handler

import (
	"context"

	"gomanager/internal/domain/user"
)

// contextKey is the type for context keys
type contextKey string

// UserContextKey is the key used to store user in context
const UserContextKey contextKey = "user"

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(ctx context.Context) *user.User {
	u, ok := ctx.Value(UserContextKey).(*user.User)
	if !ok {
		return nil
	}
	return u
}
