package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"movie-watchlist-api/internal/models"
)

const movieColumns = `m.id, m.title, m.year, m.director, m.genres, m.rating::float8,
	m.duration, m.synopsis, m.poster_url, m.cast_members`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMovie(row rowScanner) (*models.Movie, error) {
	m := &models.Movie{}
	var genres string
	if err := row.Scan(&m.ID, &m.Title, &m.Year, &m.Director, &genres, &m.Rating,
		&m.Duration, &m.Synopsis, &m.PosterURL, &m.CastMembers); err != nil {
		return nil, err
	}
	m.Genres = splitCSV(genres)
	m.Cast = splitCSV(m.CastMembers)
	return m, nil
}

func splitCSV(s string) []string {
	out := []string{}
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

type MovieRepo struct {
	db *pgxpool.Pool
}

func NewMovieRepo(db *pgxpool.Pool) *MovieRepo { return &MovieRepo{db: db} }

type MovieFilter struct {
	Genre   string
	Search  string
	Shuffle bool
}

func (r *MovieRepo) List(ctx context.Context, f MovieFilter) ([]*models.Movie, error) {

	var where []string
	var args []any

	if g := strings.TrimSpace(f.Genre); g != "" {
		args = append(args, escapeLike(strings.ToLower(g)))

		where = append(where, `(',' || REPLACE(LOWER(m.genres), ', ', ',') || ',') LIKE '%,' || $`+strconv.Itoa(len(args))+` || ',%'`)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+escapeLike(strings.ToLower(s))+"%")
		n := strconv.Itoa(len(args))
		where = append(where, `(LOWER(m.title) LIKE $`+n+` OR LOWER(m.director) LIKE $`+n+` OR LOWER(m.cast_members) LIKE $`+n+`)`)
	}

	query := `SELECT ` + movieColumns + ` FROM movies m`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	if f.Shuffle {
		query += ` ORDER BY random()`
	} else {
		query += ` ORDER BY m.id`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movies := []*models.Movie{}
	for rows.Next() {
		m, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, rows.Err()
}

func (r *MovieRepo) ListAll(ctx context.Context) ([]*models.Movie, error) {
	return r.List(ctx, MovieFilter{})
}

func (r *MovieRepo) GetByID(ctx context.Context, id int) (*models.Movie, error) {
	m, err := scanMovie(r.db.QueryRow(ctx,
		`SELECT `+movieColumns+` FROM movies m WHERE m.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
