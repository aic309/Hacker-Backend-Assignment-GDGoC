package middleware

import (
	"net/http"
	"slices"
	"strings"
)

func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := slices.Contains(allowedOrigins, "*")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && (allowAll || slices.Contains(allowedOrigins, origin)) {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {

				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join([]string{"GET", "POST", "DELETE", "OPTIONS"}, ", "))
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
