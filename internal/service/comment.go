package service

import (
	"context"
	"time"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func (s *commentService) Create(ctx context.Context, authorID uuid.UUID, dto dto.CreateCommentDto) (*model.Comment, error) {
	comment := model.Comment{
		ParentID: dto.ParentID,
		PostID: dto.PostID,
		AuthorID: authorID,
		Content: dto.Content,
	}

	createdComment, err := s.repo.Postgres.Comment.Create(ctx, comment)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create comment for post(%d): %s", comment.PostID, err.Error())
		return nil, ErrInternal
	}

	return createdComment, nil
}

func (s *commentService) FindPostComments(ctx context.Context, postID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	commentsCache, err := redisrepo.GetMany[model.FullComment](s.repo.Redis.Default, ctx, redisrepo.PostCommentsKey(postID, limit, offset))
	if err == nil {
		return commentsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get post(%d) comments from redis: %s", postID, err.Error())
		return nil, ErrInternal
	}

	comments, err := s.repo.Postgres.Comment.FindPostComments(ctx, postID, limit, offset)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get post(%d) comments from postgres: %s", postID, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.PostCommentsKey(postID, limit, offset), comments, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to set post(%d) comments in redis: %s", postID, err.Error())
		return nil, ErrInternal
	}

	return comments, nil
}

func (s *commentService) FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	repliesCache, err := redisrepo.GetMany[model.FullComment](s.repo.Redis.Default, ctx, redisrepo.CommentRepliesKey(postID, commentID, limit, offset))
	if err == nil {
		return repliesCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get comment(%d) replies from redis: %s", commentID, err.Error())
		return nil, ErrInternal
	}

	replies, err := s.repo.Postgres.Comment.FindCommentReplies(ctx, postID, commentID, limit, offset)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get comment(%d) replies from postgres: %s", commentID, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.CommentRepliesKey(postID, commentID, limit, offset), replies, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to set comment(%d) replies in redis: %s", commentID, err.Error())
		return nil, ErrInternal
	}

	return replies, nil
}

func (s *commentService) Delete(ctx context.Context, postID int64, commentID int64, authorID uuid.UUID) error {
	if err := s.repo.Postgres.Comment.Delete(ctx, postID, commentID, authorID); err != nil {
		s.logger.Sugar().Errorf("failed to delete post(%d) comment(%d): %s", postID, commentID, err.Error())
		return ErrInternal
	}

	return nil
}
