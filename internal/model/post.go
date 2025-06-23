package model

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID        int64     `json:"id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	FeedView  string    `json:"feed_view"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FullPost struct {
	Post   Post         `json:"post"`
	Author UserAuthor   `json:"author"`
	Images []*PostImage `json:"images"`
	Tags   []string     `json:"tags"`
}

type AuthorPost struct {
	Post   Post         `json:"post"`
	Images []*PostImage `json:"images"`
	Tags   []string     `json:"tags"`
}
