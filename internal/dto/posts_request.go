package dto

import "mime/multipart"

type CreatePostDto struct {
	Title   string   `form:"title" binding:"required,min=2"`
	Content string   `form:"content" binding:"required,min=20"`
	Tags    []string `form:"tags"`
}

type CreatePostImagesDto struct {
	Position   int                   `json:"position"`
	FileHeader *multipart.FileHeader `json:"file"`
}

type GetPostsDto struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
