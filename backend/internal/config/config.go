package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTExpireMinutes int
	GeminiAPIKey     string
	GeminiModel      string
	Port             string
	CORSOrigins      []string
}

func Load() (*Config, error) {

	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:  getenv("DATABASE_URL", "postgres://localhost:5432/moviedb?sslmode=disable"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  getenv("GEMINI_MODEL", "gemini-3.8-flash"),
		Port:         getenv("PORT", "8000"),
		CORSOrigins:  splitList(getenv("CORS_ORIGINS", "*")),
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters (make one with: openssl rand -hex 32)")
	}

	minutes, err := strconv.Atoi(getenv("JWT_EXPIRE_MINUTES", "60"))
	if err != nil || minutes <= 0 {
		return nil, errors.New("JWT_EXPIRE_MINUTES must be a positive number")
	}
	cfg.JWTExpireMinutes = minutes

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
