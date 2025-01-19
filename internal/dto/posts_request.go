package dto

import "mime/multipart"

type CreatePostDto struct {
	Title   string   `json:"title" binding:"required,min=2"`
	Content string   `json:"content" binding:"required,min=1"`
	Tags    []string `json:"tags"`
}

type CreatePostImagesDto struct {
	Position   int                   `json:"position"`
	FileHeader *multipart.FileHeader `json:"file"`
}
