package service

import (
	"context"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/rabbitmq"
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
	Create(ctx context.Context, authorID uuid.UUID, dto dto.CreatePostDto, images []dto.CreatePostImagesDto) (*model.Post, error)
	FindByID(ctx context.Context, id int64) (*model.FullPost, error)
	FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error)
}

type Comment interface {

}

type UserCache interface {
	CreateOrGet(ctx context.Context, id uuid.UUID, accessToken string) (*model.CachedUser, error)
	Create(ctx context.Context, cachedUser model.CachedUser) error
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.CachedUser, error)
	consumeUserUpdates(ctx context.Context)
}

type Service struct {
	Post
	UserCache
}

func New(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) *Service {
	return &Service{
		Post: newPostService(logger, repo),
		UserCache: newUserCacheService(logger, repo, rabbitmq),
	}
}

func (s *Service) StartConsumeAll(ctx context.Context) {
	go s.UserCache.consumeUserUpdates(ctx)
}
