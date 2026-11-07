package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"movie-watchlist-api/internal/auth"
)

type contextKey struct{}

func UserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(contextKey{}).(int)
	return id, ok
}

func RequireAuth(tokens *auth.TokenManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			unauthorized(w, "missing or malformed Authorization header (expected: Bearer <token>)")
			return
		}
		userID, err := tokens.Parse(strings.TrimSpace(token))
		if err != nil {
			unauthorized(w, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), contextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
