package postgres

import (
	"context"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const MAX_LIMIT = 5

func maxLimit(limit *int) {
	if *limit > MAX_LIMIT {
		*limit = MAX_LIMIT
	}
}

type Post interface {
	Create(ctx context.Context, post model.Post, tags []string) (*model.Post, error)
	FindByID(ctx context.Context, id int64) (*model.FullPost, error)
	FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error)
	FindUserNotValidatedPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AuthorPost, error)
	FindNotValidatedPosts(ctx context.Context, limit, offset int) ([]*model.FullPost, error)
	SearchByTags(ctx context.Context, tags []string, limit int, offset int) ([]*model.FullPost, error)
	IncrViews(ctx context.Context, id int64) error
	Like(ctx context.Context, postID int64, userID uuid.UUID) bool
	IncrPostLikesBy(ctx context.Context, postID, n int64) error
	Unlike(ctx context.Context, postID int64, userID uuid.UUID) bool
	IsLiked(ctx context.Context, postID int64, userID uuid.UUID) bool
	FindUserLikes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.FullPost, error)
	GetTrending(ctx context.Context, hours, limit int) ([]*model.FullPost, error)
	SearchByTitle(ctx context.Context, title string, limit, offset int) ([]*model.FullPost, error)
	Update(ctx context.Context, id int64, authorID uuid.UUID, fields map[string]interface{}) error
	UpdateValidationStatus(ctx context.Context, id int64, moderatorID uuid.UUID, validated bool, validationStatusMsg string) error
}

type Comment interface {
	Create(ctx context.Context, comment model.Comment) (*model.Comment, error)
	FindPostComments(ctx context.Context, postID int64, limit int, offset int) ([]*model.FullComment, error)
	FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error)
	Delete(ctx context.Context, postID int64, commentID int64, authorID uuid.UUID) error
	Like(ctx context.Context, commentID int64, userID uuid.UUID) bool
	IncrCommentLikesBy(ctx context.Context, commentID int64, n int64) error
	Unlike(ctx context.Context, commentID int64, userID uuid.UUID) bool
	IsLiked(ctx context.Context, commentID int64, userID uuid.UUID) bool
}

type UserCache interface {
	Create(ctx context.Context, cachedUser model.CachedUser) error
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.CachedUser, error)
}

type PostgresRepository struct {
	Post
	Comment
	UserCache
}

func New(db *pgxpool.Pool, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{
		Post: newPostRepo(db, logger),
		Comment: newCommentRepo(db, logger),
		UserCache: newUserCacheRepo(db),
	}
}
