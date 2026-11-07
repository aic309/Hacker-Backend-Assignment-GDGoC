package models

import "time"

type WatchlistItem struct {
	ID        int       `json:"id"`
	MovieID   int       `json:"movie_id"`
	Movie     *Movie    `json:"movie"`
	CreatedAt time.Time `json:"created_at"`
}
