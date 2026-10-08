package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
)

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (valid bool, userID string, err error)
}

func Auth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			valid, _, err := validator.ValidateToken(r.Context(), token)
			if err != nil {
				log.Printf("token validation failed: %v", err)
				http.Error(w, "failed to validate token", http.StatusServiceUnavailable)
				return
			}
			if !valid {
				http.Error(w, "invalid token or expired token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
