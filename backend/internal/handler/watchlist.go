package handler

import (
	"errors"
	"net/http"

	"movie-watchlist-api/internal/middleware"
	"movie-watchlist-api/internal/repository"
)

func (h *Handler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())
	items, err := h.Watchlist.List(r.Context(), userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AddToWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())

	var in struct {
		MovieIDSnake flexInt `json:"movie_id"`
		MovieIDCamel flexInt `json:"movieId"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	movieID, ok := pickMovieID(in.MovieIDSnake, in.MovieIDCamel)
	if !ok {
		writeError(w, http.StatusBadRequest, "movie_id is required and must be a positive whole number")
		return
	}

	item, err := h.Watchlist.Add(r.Context(), userID, movieID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "movie not found")
	case errors.Is(err, repository.ErrDuplicate):
		writeError(w, http.StatusConflict, "movie is already in your watchlist")
	case err != nil:
		serverError(w, r, err)
	default:
		writeJSON(w, http.StatusCreated, item)
	}
}

func (h *Handler) RemoveFromWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())
	movieID, ok := pathID(w, r, "movieId")
	if !ok {
		return
	}
	err := h.Watchlist.Remove(r.Context(), userID, movieID)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "movie is not in your watchlist")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pickMovieID(a, b flexInt) (int, bool) {
	for _, f := range []flexInt{a, b} {
		if f.Set {
			return f.Value, f.Value >= 1
		}
	}
	return 0, false
}
