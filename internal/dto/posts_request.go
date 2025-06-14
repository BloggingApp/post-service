package dto

import "github.com/google/uuid"

type CreatePostRequest struct {
	Title   string   `json:"title" binding:"required,min=2"`
	Content string   `json:"content" binding:"required,min=20"`
	Tags    []string `json:"tags"`
}

type GetPostsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type EditPostRequest struct {
	PostID   int64     `json:"id"`
	AuthorID uuid.UUID `json:"author_id"`
	Title    *string   `json:"title"`
	Content  *string   `json:"content"`
}
