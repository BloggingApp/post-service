package postgres

import (
	"context"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const MAX_LIMIT = 5

func maxLimit(limit *int) {
	if *limit > MAX_LIMIT {
		*limit = MAX_LIMIT
	}
}

type Post interface {
	Create(ctx context.Context, post model.Post, images []*model.PostImage, tags []string) (*model.Post, error)
	FindByID(ctx context.Context, id int64) (*model.FullPost, error)
	FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error)
	SearchByTags(ctx context.Context, tags []string, limit int, offset int) ([]*model.FullPost, error)
	IncrViews(ctx context.Context, id int64) error
}

type Comment interface {
	Create(ctx context.Context, comment model.Comment) (*model.Comment, error)
	FindPostComments(ctx context.Context, postID int64, limit int, offset int) ([]*model.FullComment, error)
	FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error)
	Delete(ctx context.Context, postID int64, commentID int64, authorID uuid.UUID) error
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

func New(db *pgx.Conn) *PostgresRepository {
	return &PostgresRepository{
		Post: newPostRepo(db),
		Comment: newCommentRepo(db),
		UserCache: newUserCacheRepo(db),
	}
}
