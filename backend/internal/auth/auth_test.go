package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func init() { Cost = bcrypt.MinCost }

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(hash, "password123") {
		t.Fatal("hash must not contain the password")
	}
	if !CheckPassword(hash, "password123") {
		t.Error("right password should match")
	}
	if CheckPassword(hash, "password124") {
		t.Error("wrong password should not match")
	}
	hash2, _ := HashPassword("password123")
	if hash == hash2 {
		t.Error("two hashes of the same password must differ (random salt)")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	tm := NewTokenManager("a-secret-that-is-long-enough-123456", time.Minute)
	tok, err := tm.Create(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := tm.Parse(tok)
	if err != nil || id != 42 {
		t.Fatalf("got id=%d err=%v, want 42", id, err)
	}
}

func TestTokenRejected(t *testing.T) {
	secret := "a-secret-that-is-long-enough-123456"
	tm := NewTokenManager(secret, time.Minute)
	good, _ := tm.Create(1)

	sign := func(method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))

	tests := map[string]string{
		"empty":            "",
		"garbage":          "not.a.token",
		"tampered":         good[:len(good)-2] + "xx",
		"wrong secret":     sign(jwt.SigningMethodHS256, []byte("another-secret-another-secret-12345"), jwt.RegisteredClaims{Subject: "1", ExpiresAt: future}),
		"expired":          sign(jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{Subject: "1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}),
		"no expiry":        sign(jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{Subject: "1"}),
		"other algorithm":  sign(jwt.SigningMethodHS512, []byte(secret), jwt.RegisteredClaims{Subject: "1", ExpiresAt: future}),
		"alg none":         sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.RegisteredClaims{Subject: "1", ExpiresAt: future}),
		"subject not int":  sign(jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{Subject: "abc", ExpiresAt: future}),
		"subject negative": sign(jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{Subject: "-5", ExpiresAt: future}),
	}
	for name, tok := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := tm.Parse(tok); err == nil {
				t.Errorf("token %q should be rejected", name)
			}
		})
	}
}
