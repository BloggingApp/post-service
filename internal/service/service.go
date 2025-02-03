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
	FindUserLikes(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]*model.FullPost, error)
	IsLiked(ctx context.Context, postID int64, userID uuid.UUID) bool
	Like(ctx context.Context, postID int64, userID uuid.UUID, unlike bool) error
	SchedulePostLikesUpdates()
	StartScheduledJobs()
}

type Comment interface {
	Create(ctx context.Context, authorID uuid.UUID, dto dto.CreateCommentDto) (*model.Comment, error)
	FindPostComments(ctx context.Context, postID int64, limit int, offset int) ([]*model.FullComment, error)
	FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error)
	Delete(ctx context.Context, postID int64, commentID int64, authorID uuid.UUID) error
	Like(ctx context.Context, commentID int64, userID uuid.UUID, unlike bool) error
	IsLiked(ctx context.Context, commentID int64, userID uuid.UUID) bool
	ScheduleCommentLikesUpdates()
	StartScheduledJobs()
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
	Comment
	UserCache
}

func New(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) *Service {
	return &Service{
		Post: newPostService(logger, repo),
		Comment: newCommentService(logger, repo),
		UserCache: newUserCacheService(logger, repo, rabbitmq),
	}
}

func (s *Service) StartConsumeAll(ctx context.Context) {
	go s.UserCache.consumeUserUpdates(ctx)
}

func (s *Service) StartAllScheduledJobs() {
	go s.Post.StartScheduledJobs()
	go s.Comment.StartScheduledJobs()
}
