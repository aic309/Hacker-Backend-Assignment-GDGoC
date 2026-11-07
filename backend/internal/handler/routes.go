package handler

import (
	"context"
	"net/http"

	"movie-watchlist-api/internal/auth"
	"movie-watchlist-api/internal/gemini"
	"movie-watchlist-api/internal/middleware"
	"movie-watchlist-api/internal/models"
	"movie-watchlist-api/internal/repository"
)

type Recommender interface {
	Recommend(ctx context.Context, prompt string, movies []*models.Movie) (*gemini.Recommendation, error)
}

type Handler struct {
	Users     *repository.UserRepo
	Movies    *repository.MovieRepo
	Watchlist *repository.WatchlistRepo
	Watched   *repository.WatchedRepo
	Tokens    *auth.TokenManager
	Gemini    Recommender
}

func (h *Handler) Routes(corsOrigins []string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/movies", h.ListMovies)
	mux.HandleFunc("GET /api/movies/{id}", h.GetMovie)
	mux.HandleFunc("POST /api/movies/recommend", h.Recommend)

	protected := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, middleware.RequireAuth(h.Tokens, fn))
	}
	protected("GET /api/watchlist", h.GetWatchlist)
	protected("POST /api/watchlist", h.AddToWatchlist)
	protected("DELETE /api/watchlist/{movieId}", h.RemoveFromWatchlist)
	protected("GET /api/watched", h.GetWatched)
	protected("POST /api/watched", h.LogWatched)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "route not found")
	})

	return middleware.CORS(corsOrigins, mux)
}
