package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS adds CORS headers to responses
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return CORSWithConfig(CORSConfig{
		AllowedOrigins: []string{"*"}, // Default to allow all origins
	}, next)
}

// CORSWithConfig adds CORS headers with custom configuration
func CORSWithConfig(config CORSConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Set allowed origin
		if isOriginAllowed(origin, config.AllowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
