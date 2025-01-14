package postgres

import (
	"context"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type postRepo struct {
	db *pgx.Conn
}

func newPostRepo(db *pgx.Conn) Post {
	return &postRepo{
		db: db,
	}
}

func (r *postRepo) Create(ctx context.Context, post model.Post) (*model.Post, error) {
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	post.Views = 0
	if err := r.db.QueryRow(
		ctx,
		"INSERT INTO posts(author_id, title, content, views) VALUES($1, $2, $3, $4) RETURNS id",
		post.AuthorID,
		post.Title,
		post.Content,
		post.Views,
	).Scan(&post.ID); err != nil {
		return nil, err
	}
	
	return &post, nil
}

func (r *postRepo) FindByID(ctx context.Context, id int64) (*model.FullPost, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.created_at, p.updated_at, u.username, u.display_name, u.avatar_url, h.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_hashtags h ON p.id = h.post_id
		WHERE p.id = $1`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postMap := make(map[int64]*model.FullPost)
	for rows.Next() {
		var (
			id int64
			authorID uuid.UUID
			title string
			content string
			views int64
			createdAt time.Time
			updatedAt time.Time
			username string
			displayName *string
			avatarURL *string
			hashtag *string
		)
		if err := rows.Scan(
			&id,
			&authorID,
			&title,
			&content,
			&views,
			&createdAt,
			&updatedAt,
			&username,
			&displayName,
			&avatarURL,
			&hashtag,
		); err != nil {
			return nil, err
		}

		post, exists := postMap[id]
		if !exists {
			post := &model.FullPost{
				Post: model.Post{
					ID: id,
					AuthorID: authorID,
					Title: title,
					Content: content,
					Views: views,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Author: model.UserAuthor{
					Username: username,
					DisplayName: displayName,
					AvatarURL: avatarURL,
				},
				Hashtags: make(map[string]bool),
			}
			postMap[post.Post.ID] = post
		}

		if hashtag != nil {
			postMap[post.Post.ID].Hashtags[*hashtag] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var posts []*model.FullPost
	for _, post := range postMap {
		posts = append(posts, post)
	}

	if len(posts) == 0 {
		return nil, pgx.ErrNoRows
	}

	return posts[0], nil
}

func (r *postRepo) FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.UserPost, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.created_at, p.updated_at, h.tag
		FROM posts p
		LEFT JOIN post_hashtags h ON p.id = h.post_id
		WHERE p.authorID = $1
		LIMIT $2
		OFFSET $3`,
		authorID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postsMap := make(map[int64]*model.UserPost)
	for rows.Next() {
		var (
			id int64
			authorID uuid.UUID
			title string
			content string
			views int64
			createdAt time.Time
			updatedAt time.Time
			hashtag *string
		)
		if err := rows.Scan(
			&id,
			&authorID,
			&title,
			&content,
			&views,
			&createdAt,
			&updatedAt,
			&hashtag,
		); err != nil {
			return nil, err
		}

		post, exists := postsMap[id]
		if !exists {
			post = &model.UserPost{
				Post: model.Post{
					ID: id,
					AuthorID: authorID,
					Title: title,
					Content: content,
					Views: views,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Hashtags: make(map[string]bool),
			}
			postsMap[post.Post.ID] = post
		}

		if hashtag != nil {
			post.Hashtags[*hashtag] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var posts []*model.UserPost
	for _, post := range postsMap {
		posts = append(posts, post)
	}

	return posts, nil
}
