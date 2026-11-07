package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"movie-watchlist-api/internal/gemini"
	"movie-watchlist-api/internal/repository"
)

func (h *Handler) ListMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	movies, err := h.Movies.List(r.Context(), repository.MovieFilter{
		Genre:   q.Get("genre"),
		Search:  q.Get("search"),
		Shuffle: q.Get("shuffle") == "true" || q.Get("shuffle") == "1",
	})
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, movies)
}

func (h *Handler) GetMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	movie, err := h.Movies.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "movie not found")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, movie)
}

func (h *Handler) Recommend(w http.ResponseWriter, r *http.Request) {

	var in struct {
		Prompt      string `json:"prompt"`
		Query       string `json:"query"`
		Description string `json:"description"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(in.Query)
	}
	if prompt == "" {
		prompt = strings.TrimSpace(in.Description)
	}
	if prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	if utf8.RuneCountInString(prompt) > 500 {
		writeError(w, http.StatusBadRequest, "prompt must be at most 500 characters")
		return
	}

	movies, err := h.Movies.ListAll(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}

	rec, err := h.Gemini.Recommend(r.Context(), prompt, movies)
	switch {
	case errors.Is(err, gemini.ErrNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "AI recommendation is not available (GEMINI_API_KEY is not set on the server)")
		return
	case err != nil:

		serverLog(r, err)
		writeError(w, http.StatusBadGateway, "the AI service failed, please try again")
		return
	}

	movie, err := h.Movies.GetByID(r.Context(), rec.MovieID)
	if err != nil {
		serverLog(r, err)
		writeError(w, http.StatusBadGateway, "the AI service failed, please try again")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"movie_id":  movie.ID,
		"reasoning": rec.Reasoning,
		"movie":     movie,
	})
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, name+" must be a positive whole number")
		return 0, false
	}
	return id, true
}
