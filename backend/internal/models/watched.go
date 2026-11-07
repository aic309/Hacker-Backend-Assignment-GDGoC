package models

import "time"

type WatchedItem struct {
	ID        int       `json:"id"`
	MovieID   int       `json:"movie_id"`
	Rating    int       `json:"rating"`
	Review    string    `json:"review"`
	Movie     *Movie    `json:"movie"`
	CreatedAt time.Time `json:"created_at"`
}
