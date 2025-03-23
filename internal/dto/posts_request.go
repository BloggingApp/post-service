package dto

import "mime/multipart"

type CreatePostRequest struct {
	Title   string   `form:"title" binding:"required,min=2"`
	Content string   `form:"content" binding:"required,min=20"`
	Tags    []string `form:"tags"`
}

type CreatePostImagesRequest struct {
	Position   int                   `json:"position"`
	FileHeader *multipart.FileHeader `json:"file"`
}

type GetPostsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
