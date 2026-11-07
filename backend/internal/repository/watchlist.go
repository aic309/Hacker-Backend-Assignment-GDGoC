package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"movie-watchlist-api/internal/models"
)

type WatchlistRepo struct {
	db *pgxpool.Pool
}

func NewWatchlistRepo(db *pgxpool.Pool) *WatchlistRepo { return &WatchlistRepo{db: db} }

func (r *WatchlistRepo) List(ctx context.Context, userID int) ([]*models.WatchlistItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT w.id, w.movie_id, w.created_at, `+movieColumns+`
		 FROM watchlist w
		 JOIN movies m ON m.id = w.movie_id
		 WHERE w.user_id = $1
		 ORDER BY w.created_at DESC, w.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*models.WatchlistItem{}
	for rows.Next() {
		it, err := scanWatchlist(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func scanWatchlist(row rowScanner) (*models.WatchlistItem, error) {
	it := &models.WatchlistItem{}
	m := &models.Movie{}
	var genres string
	err := row.Scan(&it.ID, &it.MovieID, &it.CreatedAt,
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

func (r *WatchlistRepo) Add(ctx context.Context, userID, movieID int) (*models.WatchlistItem, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO watchlist (user_id, movie_id) VALUES ($1, $2) RETURNING id`,
		userID, movieID).Scan(&id)
	if hasCode(err, pgUniqueViolation) {
		return nil, ErrDuplicate
	}
	if hasCode(err, pgForeignKeyViolation) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return scanWatchlist(r.db.QueryRow(ctx,
		`SELECT w.id, w.movie_id, w.created_at, `+movieColumns+`
		 FROM watchlist w JOIN movies m ON m.id = w.movie_id
		 WHERE w.id = $1`, id))
}

func (r *WatchlistRepo) Remove(ctx context.Context, userID, movieID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM watchlist WHERE user_id = $1 AND movie_id = $2`, userID, movieID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
