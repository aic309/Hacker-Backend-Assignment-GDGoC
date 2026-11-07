package models

type Movie struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	Director string   `json:"director"`
	Genres   []string `json:"genres"`
	Rating   float64  `json:"rating"`
	Duration string   `json:"duration"`
	Synopsis string   `json:"synopsis"`

	PosterURL string `json:"poster_url"`

	Cast        []string `json:"cast"`
	CastMembers string   `json:"cast_members"`
}
