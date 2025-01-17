package service

import (
	"github.com/BloggingApp/post-service/internal/repository"
	"go.uber.org/zap"
)

type commentService struct {
	logger *zap.Logger
	repo *repository.Repository
}

func newCommentService(logger *zap.Logger, repo *repository.Repository) Comment {
	return &commentService{
		logger: logger,
		repo: repo,
	}
}
