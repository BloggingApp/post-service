package service

import (
	"context"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type postService struct {
	logger *zap.Logger
	repo *repository.Repository
}

func newPostService(logger *zap.Logger, repo *repository.Repository) Post {
	return &postService{
		logger: logger,
		repo: repo,
	}
}

func (s *postService) Create(ctx context.Context, post model.Post, images []*model.PostImage) (*model.Post, error) {
	createdPost, err := s.repo.Postgres.Post.Create(ctx, post, images)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user(%s) post: %s", post.AuthorID.String(), err.Error())
		return nil, ErrInternal
	}

	return createdPost, nil
}

func (s *postService) FindByID(ctx context.Context, id int64) (*model.FullPost, error) {
	cachedPost, err := redisrepo.Get[model.FullPost](s.repo.Redis.Default, ctx, redisrepo.PostKey(id))
	if err == nil {
		return cachedPost, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get post(%d) from redis: %s", id, err.Error())
		return nil, err
	}

	post, err := s.repo.Postgres.Post.FindByID(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Sugar().Errorf("failed to find post(%d) from postgres: %s", id, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.PostKey(id), post, time.Hour); err != nil {
		s.logger.Sugar().Errorf("failed to set post(%d) in redis: %s", id, err.Error())
		return nil, ErrInternal
	}

	go func(ctx context.Context, s *postService, postID int64) {
		if err := s.repo.Postgres.Post.IncrViews(ctx, id); err != nil {
			s.logger.Sugar().Errorf("failed to increment views for post(%d): %s", postID, err.Error())
		}
	}(ctx, s, id)

	return post, nil
}

func (s *postService) FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error) {
	maxLimit(&limit)

	cachedPosts, err := redisrepo.GetMany[model.AuthorPost](s.repo.Redis.Default, ctx, redisrepo.AuthorPostsKey(authorID.String(), limit, offset))
	if err == nil {
		return cachedPosts, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get author(%s) posts from redis: %s", authorID.String(), err.Error())
		return nil, ErrInternal
	}

	posts, err := s.repo.Postgres.Post.FindAuthorPosts(ctx, authorID, limit, offset)
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Sugar().Errorf("failed to find author(%s) posts from postgres: %s", authorID.String(), err.Error())
		return nil, err
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.AuthorPostsKey(authorID.String(), limit, offset), posts, time.Hour); err != nil {
		s.logger.Sugar().Errorf("failed to set author(%s) posts in redis: %s", authorID.String(), err.Error())
		return nil, err
	}

	return posts, nil
}
