package database

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The SQL files are packed inside the program when it is built,
// so the program does not need to find them on disk later.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cannot reach PostgreSQL (is it running? is DATABASE_URL right?): %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, f := range files {
		sql, err := migrationFiles.ReadFile("migrations/" + f.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", f.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("run %s: %w", f.Name(), err)
		}
	}
	return nil
}
