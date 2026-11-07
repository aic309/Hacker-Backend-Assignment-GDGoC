package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const MaxPasswordBytes = 72

var Cost = bcrypt.DefaultCost

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func WasteTime(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
