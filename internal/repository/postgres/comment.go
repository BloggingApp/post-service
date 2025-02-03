package postgres

import (
	"context"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type commentRepo struct {
	db *pgxpool.Pool
	logger *zap.Logger
}

func newCommentRepo(db *pgxpool.Pool, logger *zap.Logger) Comment {
	return &commentRepo{
		db: db,
		logger: logger,
	}
}

func (r *commentRepo) Create(ctx context.Context, comment model.Comment) (*model.Comment, error) {
	comment.CreatedAt = time.Now()
	comment.Likes = 0
	if err := r.db.QueryRow(
		ctx,
		"INSERT INTO comments(parent_id, post_id, author_id, content, likes) VALUES($1, $2, $3, $4, $5) RETURNING id",
		comment.ParentID,
		comment.PostID,
		comment.AuthorID,
		comment.Content,
		comment.Likes,
	).Scan(&comment.ID); err != nil {
		return nil, err
	}

	return &comment, nil
}

func (r *commentRepo) FindPostComments(ctx context.Context, postID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		c.id, c.post_id, c.author_id, c.content, c.likes, c.created_at, u.username, u.display_name, u.avatar_url
		FROM comments c
		JOIN cached_users u ON c.author_id = u.id
		WHERE c.post_id = $1 AND c.parent_id IS NULL
		ORDER BY c.likes DESC, c.created_at DESC
		LIMIT $2
		OFFSET $3`,
		postID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.FullComment
	for rows.Next() {
		var comment model.FullComment
		if err := rows.Scan(
			&comment.Comment.ID,
			&comment.Comment.PostID,
			&comment.Comment.AuthorID,
			&comment.Comment.Content,
			&comment.Comment.Likes,
			&comment.Comment.CreatedAt,
			&comment.Author.Username,
			&comment.Author.DisplayName,
			&comment.Author.AvatarURL,
		); err != nil {
			return nil, err
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *commentRepo) FindCommentReplies(ctx context.Context, postID int64, commentID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		c.id, c.parent_id, c.post_id, c.author_id, c.content, c.likes, c.created_at, u.username, u.display_name, u.avatar_url
		FROM comments c
		JOIN cached_users u ON c.author_id = u.id
		WHERE c.post_id = $1 AND c.parent_id = $2
		ORDER BY c.likes DESC, c.created_at DESC
		LIMIT $3
		OFFSET $4`,
		postID,
		commentID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.FullComment
	for rows.Next() {
		var comment model.FullComment
		if err := rows.Scan(
			&comment.Comment.ID,
			&comment.Comment.ParentID,
			&comment.Comment.PostID,
			&comment.Comment.AuthorID,
			&comment.Comment.Content,
			&comment.Comment.Likes,
			&comment.Comment.CreatedAt,
			&comment.Author.Username,
			&comment.Author.DisplayName,
			&comment.Author.AvatarURL,
		); err != nil {
			return nil, err
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *commentRepo) Delete(ctx context.Context, postID int64, commentID int64, authorID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM comments WHERE post_id = $1 AND id = $2 AND author_id = $3", postID, commentID, authorID)
	return err
}

func (r *commentRepo) Like(ctx context.Context, commentID int64, userID uuid.UUID) bool {
	cmd, err := r.db.Exec(ctx, "INSERT INTO comment_likes(comment_id, user_id) VALUES($1, $2) ON CONFLICT DO NOTHING", commentID, userID)
	return err == nil && cmd.RowsAffected() == 1
}

func (r *commentRepo) IncrCommentLikesBy(ctx context.Context, commentID int64, n int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE comments
		SET likes = CASE
			WHEN likes + $1 >= 0 THEN likes + $1
			ELSE 0
		END
		WHERE id = $2
	`, n, commentID)
	return err
}

func (r *commentRepo) Unlike(ctx context.Context, commentID int64, userID uuid.UUID) bool {
	cmd, err := r.db.Exec(ctx, "DELETE FROM comment_likes WHERE comment_id = $1 AND user_id = $2", commentID, userID)
	return err == nil && cmd.RowsAffected() == 1
}

func (r *commentRepo) IsLiked(ctx context.Context, commentID int64, userID uuid.UUID) bool {
	var exists bool
	var err error

	for retries := 0; retries < 3; retries++ {
		err = r.db.QueryRow(ctx, "SELECT count(*) > 0 FROM comment_likes WHERE comment_id = $1 AND user_id = $2", commentID, userID).Scan(&exists)
		if err == nil {
			break
		}
		if err.Error() == "conn busy" {
			time.Sleep(time.Duration(retries+1) * time.Millisecond * 100)
			continue
		}
		break
	}

	if err != nil {
		r.logger.Sugar().Errorf("failed to get is liked for user(%s): %s", userID.String(), err.Error())
		return false
	}

	return exists
}
