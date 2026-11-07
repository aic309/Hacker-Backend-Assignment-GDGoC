package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"

	"movie-watchlist-api/internal/config"
	"movie-watchlist-api/internal/database"
)

type movieJSON struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Year      int      `json:"year"`
	Director  string   `json:"director"`
	Genres    []string `json:"genres"`
	Rating    float64  `json:"rating"`
	Duration  string   `json:"duration"`
	Synopsis  string   `json:"synopsis"`
	PosterURL string   `json:"poster_url"`
	Cast      []string `json:"cast"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run() error {
	path, err := findFile()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var movies []movieJSON
	if err := json.Unmarshal(raw, &movies); err != nil {
		return fmt.Errorf("%s is not a JSON list of movies: %w", path, err)
	}
	for i, m := range movies {
		if m.ID < 1 || strings.TrimSpace(m.Title) == "" {
			return fmt.Errorf("movie number %d in %s has no id or no title", i+1, path)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, m := range movies {
		if err := upsert(ctx, tx, m); err != nil {
			return fmt.Errorf("movie %d (%s): %w", m.ID, m.Title, err)
		}
	}

	if _, err := tx.Exec(ctx,
		`SELECT setval(pg_get_serial_sequence('movies', 'id'), (SELECT MAX(id) FROM movies))`); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("Seeded %d movies from %s\n", len(movies), path)
	return nil
}

func upsert(ctx context.Context, tx pgx.Tx, m movieJSON) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO movies (id, title, year, director, genres, rating, duration, synopsis, poster_url, cast_members)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title, year = EXCLUDED.year, director = EXCLUDED.director,
			genres = EXCLUDED.genres, rating = EXCLUDED.rating, duration = EXCLUDED.duration,
			synopsis = EXCLUDED.synopsis, poster_url = EXCLUDED.poster_url,
			cast_members = EXCLUDED.cast_members`,
		m.ID, m.Title, m.Year, m.Director, strings.Join(m.Genres, ", "), m.Rating,
		m.Duration, m.Synopsis, m.PosterURL, strings.Join(m.Cast, ", "))
	return err
}

func findFile() (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}
	for _, p := range []string{"data/movies.json", "../data/movies.json"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("movies.json not found. Run from the project folder, or give the path: go run ./cmd/seed path/to/movies.json")
}
