package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"movie-watchlist-api/internal/middleware"
	"movie-watchlist-api/internal/repository"
)

func (h *Handler) GetWatched(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())
	items, err := h.Watched.List(r.Context(), userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) LogWatched(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())

	var in struct {
		MovieIDSnake flexInt `json:"movie_id"`
		MovieIDCamel flexInt `json:"movieId"`
		Rating       flexInt `json:"rating"`
		Review       string  `json:"review"`
		Opinion      string  `json:"opinion"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	movieID, ok := pickMovieID(in.MovieIDSnake, in.MovieIDCamel)
	if !ok {
		writeError(w, http.StatusBadRequest, "movie_id is required and must be a positive whole number")
		return
	}
	if !in.Rating.Set || in.Rating.Value < 1 || in.Rating.Value > 5 {
		writeError(w, http.StatusBadRequest, "rating is required and must be a whole number from 1 to 5")
		return
	}
	review := strings.TrimSpace(in.Review)
	if review == "" {
		review = strings.TrimSpace(in.Opinion)
	}
	if utf8.RuneCountInString(review) > 2000 {
		writeError(w, http.StatusBadRequest, "review must be at most 2000 characters")
		return
	}

	item, err := h.Watched.Add(r.Context(), userID, movieID, in.Rating.Value, review)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "movie not found")
	case errors.Is(err, repository.ErrDuplicate):
		writeError(w, http.StatusConflict, "you already logged this movie as watched")
	case err != nil:
		serverError(w, r, err)
	default:
		writeJSON(w, http.StatusCreated, item)
	}
}
