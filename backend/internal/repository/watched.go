package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"movie-watchlist-api/internal/models"
)

type WatchedRepo struct {
	db *pgxpool.Pool
}

func NewWatchedRepo(db *pgxpool.Pool) *WatchedRepo { return &WatchedRepo{db: db} }

const watchedSelect = `SELECT d.id, d.movie_id, d.rating, d.review, d.created_at, ` + movieColumns + `
	FROM watched d JOIN movies m ON m.id = d.movie_id `

func scanWatched(row rowScanner) (*models.WatchedItem, error) {
	it := &models.WatchedItem{}
	m := &models.Movie{}
	var genres string
	err := row.Scan(&it.ID, &it.MovieID, &it.Rating, &it.Review, &it.CreatedAt,
		&m.ID, &m.Title, &m.Year, &m.Director, &genres, &m.Rating,
		&m.Duration, &m.Synopsis, &m.PosterURL, &m.CastMembers)
	if err != nil {
		return nil, err
	}
	m.Genres = splitCSV(genres)
	m.Cast = splitCSV(m.CastMembers)
	it.Movie = m
	return it, nil
}

func (r *WatchedRepo) List(ctx context.Context, userID int) ([]*models.WatchedItem, error) {
	rows, err := r.db.Query(ctx,
		watchedSelect+`WHERE d.user_id = $1 ORDER BY d.created_at DESC, d.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*models.WatchedItem{}
	for rows.Next() {
		it, err := scanWatched(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *WatchedRepo) Add(ctx context.Context, userID, movieID, rating int, review string) (*models.WatchedItem, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO watched (user_id, movie_id, rating, review)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, movieID, rating, review).Scan(&id)
	if hasCode(err, pgUniqueViolation) {
		return nil, ErrDuplicate
	}
	if hasCode(err, pgForeignKeyViolation) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return scanWatched(r.db.QueryRow(ctx, watchedSelect+`WHERE d.id = $1`, id))
}
