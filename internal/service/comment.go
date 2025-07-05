package service

import (
	"context"
	"fmt"
	"time"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type commentService struct {
	logger *zap.Logger
	repo *repository.Repository
	rdb *redis.Client
	scheduler gocron.Scheduler
}

func newCommentService(logger *zap.Logger, repo *repository.Repository, rdb *redis.Client) Comment {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	return &commentService{
		logger: logger,
		repo: repo,
		rdb: rdb,
		scheduler: scheduler,
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

	commentsCache, err := redisrepo.GetMany[model.FullComment](s.rdb, ctx, redisrepo.PostCommentsKey(postID, limit, offset))
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

	if err := redisrepo.SetJSON(s.rdb, ctx, redisrepo.PostCommentsKey(postID, limit, offset), comments, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to set post(%d) comments in redis: %s", postID, err.Error())
		return nil, ErrInternal
	}

	return comments, nil
}

func (s *commentService) FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	repliesCache, err := redisrepo.GetMany[model.FullComment](s.rdb, ctx, redisrepo.CommentRepliesKey(postID, commentID, limit, offset))
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

	if err := redisrepo.SetJSON(s.rdb, ctx, redisrepo.CommentRepliesKey(postID, commentID, limit, offset), replies, time.Minute); err != nil {
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

// Set 'unlike' value to true if you want to UNLIKE a comment
func (s *commentService) Like(ctx context.Context, commentID int64, userID uuid.UUID, unlike bool) error {
	var affected bool
	var delta int64
	if unlike {
		affected = s.repo.Postgres.Comment.Unlike(ctx, commentID, userID)
		delta = -1
	} else {
		affected = s.repo.Postgres.Comment.Like(ctx, commentID, userID)
		delta = 1
	}

	if !affected {
		return ErrFailedToLikeTheComment
	}

	// Update "is liked" cache
	if err := s.rdb.Set(ctx, redisrepo.IsLikedCommentKey(userID.String(), commentID), !unlike, time.Minute).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) is liked for comment(%d) in redis: %s", userID.String(), commentID, err.Error())
		return ErrInternal
	}

	if err := s.updateCommentCachedLikes(ctx, commentID, delta); err != nil {
		return err
	}

	return nil
}

func (s *commentService) IsLiked(ctx context.Context, commentID int64, userID uuid.UUID) bool {
	isLikedCache, err := s.rdb.Get(ctx, redisrepo.IsLikedCommentKey(userID.String(), commentID)).Bool()
	if err == nil {
		return isLikedCache
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get user(%s) is liked comment(%d) value from redis: %s", userID.String(), commentID, err.Error())
		return false
	}

	isLiked := s.repo.Postgres.Comment.IsLiked(ctx, commentID, userID)

	if err := s.rdb.Set(ctx, redisrepo.IsLikedCommentKey(userID.String(), commentID), isLiked, time.Minute).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) is liked comment(%d) value in redis: %s", userID.String(), commentID, err.Error())
		return false
	}

	return isLiked
}

func (s *commentService) updateCommentCachedLikes(ctx context.Context, commentID int64, delta int64) error {
	key := redisrepo.CommentLikesKey(commentID)

	if err := s.rdb.IncrBy(ctx, key, delta).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to increment key(%s) in redis: %s", key, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *commentService) commentsBatchLikesUpdate(ctx context.Context) error {
	commentKeys, err := s.rdb.Keys(ctx, redisrepo.COMMENT_LIKES_KEY_PATTERN).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get keys with pattern(%s) from redis: %s", redisrepo.COMMENT_LIKES_KEY_PATTERN, err.Error())
	}
	if err == redis.Nil || len(commentKeys) == 0 {
		return nil
	}

	for _, commentKey := range commentKeys {
		// Get comment ID frmo redis key
		commentID, err := redisrepo.GetCommentIDFromCommentLikesKey(commentKey)
		if err != nil {
			continue
		}

		// Get comment's cached likes
		n, err := s.rdb.Get(ctx, commentKey).Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to get comment(%d) cached likes from redis: %s", commentID, err.Error())
		}
		if err == redis.Nil {
			continue
		}

		if err := s.repo.Postgres.Comment.IncrCommentLikesBy(ctx, commentID, n); err != nil {
			return fmt.Errorf("failed to incr comment(%d) likes by(%d): %s", commentID, n, err.Error())
		}

		if err := s.rdb.Del(ctx, commentKey).Err(); err != nil {
			return fmt.Errorf("failed to delete comment(%d) likes from redis: %s", commentID, err.Error())
		}
	}

	return nil
}

func (s *commentService) ScheduleCommentLikesUpdates() {
	s.scheduler.NewJob(gocron.DurationJob(COMMENT_LIKES_UPDATE_TIMEOUT), gocron.NewTask(func(ctx context.Context) {
		if err := s.commentsBatchLikesUpdate(ctx); err != nil {
			s.logger.Sugar().Error(err.Error())
		}
	}))
}

func (s *commentService) StartScheduledJobs() {
	s.ScheduleCommentLikesUpdates()

	s.scheduler.Start()
}
