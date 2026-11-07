package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"movie-watchlist-api/internal/auth"
	"movie-watchlist-api/internal/config"
	"movie-watchlist-api/internal/database"
	"movie-watchlist-api/internal/gemini"
	"movie-watchlist-api/internal/handler"
	"movie-watchlist-api/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	h := &handler.Handler{
		Users:     repository.NewUserRepo(pool),
		Movies:    repository.NewMovieRepo(pool),
		Watchlist: repository.NewWatchlistRepo(pool),
		Watched:   repository.NewWatchedRepo(pool),
		Tokens:    auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTExpireMinutes)*time.Minute),
		Gemini:    gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel),
	}
	if cfg.GeminiAPIKey == "" {
		log.Println("warning: GEMINI_API_KEY is empty, POST /api/movies/recommend will answer 503")
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: h.Routes(cfg.CORSOrigins),

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("server running on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}
