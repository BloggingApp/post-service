package service

import (
	"context"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const MAX_LIMIT = 5

func maxLimit(limit *int) {
	if *limit > MAX_LIMIT {
		*limit = MAX_LIMIT
	}
}

type Post interface {
	Create(ctx context.Context, post model.Post) (*model.Post, error)
	FindByID(ctx context.Context, id int64) (*model.FullPost, error)
	FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error)
}

type Comment interface {

}

type Service struct {
	Post
}

func New(logger *zap.Logger, repo *repository.Repository) *Service {
	return &Service{
		Post: newPostService(logger, repo),
	}
}
