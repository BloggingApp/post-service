package dto

import "github.com/BloggingApp/post-service/internal/model"

type GetPost struct {
	Post model.FullPost `json:"post"`
	IsLiked bool `json:"is_liked"`
}
