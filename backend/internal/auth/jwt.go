package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

func (t *TokenManager) Create(userID int) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *TokenManager) Parse(token string) (int, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },

		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),

		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return 0, ErrInvalidToken
	}
	id, err := strconv.Atoi(claims.Subject)
	if err != nil || id <= 0 {
		return 0, ErrInvalidToken
	}
	return id, nil
}
