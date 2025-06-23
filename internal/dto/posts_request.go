package dto

import "github.com/google/uuid"

type CreatePostRequest struct {
	Title   string   `json:"title" binding:"required,min=2"`
	Content string   `json:"content" binding:"required,min=100,max=15000"`
	FeedView string `json:"feed_view" binding:"required,min=100,max=2000"`
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
	FeedView *string   `json:"feed_view"`
}
