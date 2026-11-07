package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"movie-watchlist-api/internal/auth"
)

func TestRequireAuth(t *testing.T) {
	tm := auth.NewTokenManager("a-secret-that-is-long-enough-123456", time.Minute)
	token, _ := tm.Create(7)

	var seenID int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID, _ = UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := RequireAuth(tm, next)

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", 401},
		{"wrong scheme", "Basic " + token, 401},
		{"no token after Bearer", "Bearer ", 401},
		{"garbage token", "Bearer abc.def.ghi", 401},
		{"valid token", "Bearer " + token, 200},
		{"scheme is case-insensitive", "bearer " + token, 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seenID = 0
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			if tc.want == 401 && seenID != 0 {
				t.Error("handler must not run on 401")
			}
			if tc.want == 200 && seenID != 7 {
				t.Errorf("user id in context = %d, want 7", seenID)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })

	t.Run("preflight is answered without reaching the handler", func(t *testing.T) {
		reached = false
		h := CORS([]string{"http://localhost:3000"}, next)
		req := httptest.NewRequest(http.MethodOptions, "/api/watchlist", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent || reached {
			t.Fatalf("status=%d reached=%v", rec.Code, reached)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Error("missing allow-origin")
		}
		if rec.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("missing allow-headers")
		}
	})
	t.Run("unknown origin gets no CORS headers", func(t *testing.T) {
		h := CORS([]string{"http://localhost:3000"}, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://evil.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("evil origin must not be allowed")
		}
	})
	t.Run("star allows any origin", func(t *testing.T) {
		h := CORS([]string{"*"}, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://anything.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("want *")
		}
	})
}
