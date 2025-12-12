package middleware

import (
	"context"
	"net/http"
	"strings"

	"gomanager/internal/application/auth"
	"gomanager/internal/delivery/http/handler"
	"gomanager/internal/domain/user"
)

// Auth middleware validates the authorization token
func Auth(authService auth.Service) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				handler.SendError(w, "Authorization required", http.StatusUnauthorized)
				return
			}

			u, err := authService.ValidateToken(token)
			if err != nil {
				handler.SendError(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), handler.UserContextKey, u)
			next(w, r.WithContext(ctx))
		}
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(roles ...user.Role) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u := GetUserFromContext(r.Context())
			if u == nil {
				handler.SendError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, role := range roles {
				if u.Role == role {
					next(w, r)
					return
				}
			}

			handler.SendError(w, "Insufficient permissions", http.StatusForbidden)
		}
	}
}

// OptionalAuth middleware adds user to context if token is valid, but doesn't require it
func OptionalAuth(authService auth.Service) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				if u, err := authService.ValidateToken(token); err == nil {
					ctx := context.WithValue(r.Context(), handler.UserContextKey, u)
					r = r.WithContext(ctx)
				}
			}
			next(w, r)
		}
	}
}

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(ctx context.Context) *user.User {
	return handler.GetUserFromContext(ctx)
}

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Check query parameter (for downloads)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}
