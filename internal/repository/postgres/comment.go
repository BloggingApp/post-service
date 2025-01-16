package postgres

import (
	"context"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type commentRepo struct {
	db *pgx.Conn
}

func newCommentRepo(db *pgx.Conn) Comment {
	return &commentRepo{
		db: db,
	}
}

func (r *commentRepo) Create(ctx context.Context, comment model.Comment) (*model.Comment, error) {
	comment.CreatedAt = time.Now()
	comment.Likes = 0
	if err := r.db.QueryRow(
		ctx,
		"INSERT INTO comments(parent_id, post_id, author_id, content, likes) VALUES($1, $2, $3, $4, $5) RETURNS id",
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
		WHERE c.post_id = $1
		LIMIT $2
		OFFSET $3
		ORDER BY c.likes DESC
		ORDER BY c.created_at DESC`,
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

func (r *commentRepo) FindCommentReplies(ctx context.Context, commentID int64, limit int, offset int) ([]*model.FullComment, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		c.id, c.parent_id, c.post_id, c.author_id, c.content, c.likes, c.created_at, u.username, u.display_name, u.avatar_url
		FROM comments c
		JOIN cached_users u ON c.author_id = u.id
		WHERE c.parent_id = $1
		LIMIT $2
		OFFSET $3
		ORDER BY c.likes DESC
		ORDER BY c.created_at DESC`,
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

func (r *commentRepo) Delete(ctx context.Context, commentID int64, authorID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM comments WHERE id = $1 AND author_id = $2", commentID, authorID)
	return err
}
