package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"movie-watchlist-api/internal/auth"
	"movie-watchlist-api/internal/repository"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Username string `json:"username"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	email := strings.TrimSpace(in.Email)
	if !validEmail(email) {
		writeError(w, http.StatusBadRequest, "email is not valid")
		return
	}
	if utf8.RuneCountInString(in.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if len(in.Password) > auth.MaxPasswordBytes {
		writeError(w, http.StatusBadRequest, "password must be at most 72 bytes")
		return
	}
	username := strings.TrimSpace(in.Username)
	if username == "" {
		username, _, _ = strings.Cut(email, "@")
	}
	if utf8.RuneCountInString(username) > 50 {
		writeError(w, http.StatusBadRequest, "username must be at most 50 characters")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverError(w, r, err)
		return
	}
	user, err := h.Users.Create(r.Context(), username, email, hash)
	if errors.Is(err, repository.ErrDuplicate) {
		writeError(w, http.StatusConflict, "email is already registered")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "registration successful, please log in",
		"user":    userView(user.ID, user.Username, user.Email),
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Email) == "" || in.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	const badLogin = "invalid email or password"

	user, err := h.Users.FindByEmail(r.Context(), strings.TrimSpace(in.Email))
	if errors.Is(err, repository.ErrNotFound) {
		auth.WasteTime(in.Password)
		writeError(w, http.StatusUnauthorized, badLogin)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !auth.CheckPassword(user.PasswordHash, in.Password) {
		writeError(w, http.StatusUnauthorized, badLogin)
		return
	}

	token, err := h.Tokens.Create(user.ID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  userView(user.ID, user.Username, user.Email),
	})
}

func userView(id int, username, email string) map[string]any {
	return map[string]any{"id": id, "username": username, "email": email}
}

func validEmail(s string) bool {
	if s == "" || len(s) > 255 {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}

	_, domain, _ := strings.Cut(s, "@")
	return strings.Contains(domain, ".")
}
