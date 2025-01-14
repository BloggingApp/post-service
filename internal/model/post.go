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
	Views     int64     `json:"views"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FullPost struct {
	Post     Post            `json:"post"`
	Author   UserAuthor      `json:"author"`
	Hashtags map[string]bool `json:"hashtags"`
}

type UserPost struct {
	Post     Post            `json:"post"`
	Hashtags map[string]bool `json:"hashtags"`
}
