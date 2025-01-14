package model

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        int64     `json:"id"`
	ParentID  *int64    `json:"parent_id"`
	PostID    int64     `json:"post_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	Likes     int64     `json:"likes"`
	CreatedAt time.Time `json:"created_at"`
}

type FullComment struct {
	Comment Comment    `json:"comment"`
	Author  UserAuthor `json:"author"`
}
