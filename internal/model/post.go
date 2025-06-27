package model

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID              int64     `json:"id"`
	AuthorID        uuid.UUID `json:"author_id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	FeedView        string    `json:"feed_view"`
	Views           int64     `json:"views"`
	Likes           int64     `json:"likes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Validated       bool      `json:"validated"`
	NotValidatedMsg *string   `json:"not_validated_msg"`
}

type FullPost struct {
	Post   Post         `json:"post"`
	Author UserAuthor   `json:"author"`
	Tags   []string     `json:"tags"`
}

type AuthorPost struct {
	Post   Post         `json:"post"`
	Tags   []string     `json:"tags"`
}
