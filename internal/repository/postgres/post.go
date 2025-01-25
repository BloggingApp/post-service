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

func (r *postRepo) Create(ctx context.Context, post model.Post, images []*model.PostImage, tags []string) (*model.Post, error) {
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	post.Views = 0
	post.Likes = 0

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(
		ctx,
		"INSERT INTO posts(author_id, title, content, views, likes) VALUES($1, $2, $3, $4, $5) RETURNING id",
		post.AuthorID,
		post.Title,
		post.Content,
		post.Views,
		post.Likes,
	).Scan(&post.ID); err != nil {
		return nil, err
	}

	for _, img := range images {
		_, err := tx.Exec(ctx, "INSERT INTO post_images(post_id, url, position) VALUES($1, $2, $3)", post.ID, img.URL, img.Position)
		if err != nil {
			return nil, err
		}
	}

	for _, tag := range tags {
		_, err := tx.Exec(ctx, "INSERT INTO post_tags(post_id, tag) VALUES($1, $2)", post.ID, tag)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *postRepo) FindByID(ctx context.Context, id int64) (*model.FullPost, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at, u.username, u.display_name, u.avatar_url, i.url, i.position, t.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
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
			likes int64
			createdAt time.Time
			updatedAt time.Time
			username string
			displayName *string
			avatarURL *string
			imageURL *string
			imagePosition *int
			tag *string
		)
		if err := rows.Scan(
			&id,
			&authorID,
			&title,
			&content,
			&views,
			&likes,
			&createdAt,
			&updatedAt,
			&username,
			&displayName,
			&avatarURL,
			&imageURL,
			&imagePosition,
			&tag,
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
				Images: []*model.PostImage{},
				Tags: []string{},
			}
			postMap[post.Post.ID] = post
		}

		if imageURL != nil && imagePosition != nil {
			postMap[post.Post.ID].Images = append(postMap[post.Post.ID].Images, &model.PostImage{URL: *imageURL, Position: *imagePosition})
		}

		if tag != nil {
			postMap[post.Post.ID].Tags = append(postMap[post.Post.ID].Tags, *tag)
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

func (r *postRepo) FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at, i.url, i.position, t.tag
		FROM posts p
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE p.author_id = $1
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

	postsMap := make(map[int64]*model.AuthorPost)
	for rows.Next() {
		var (
			id int64
			authorID uuid.UUID
			title string
			content string
			views int64
			likes int64
			createdAt time.Time
			updatedAt time.Time
			imageURL *string
			imagePosition *int
			tag *string
		)
		if err := rows.Scan(
			&id,
			&authorID,
			&title,
			&content,
			&views,
			&likes,
			&createdAt,
			&updatedAt,
			&imageURL,
			&imagePosition,
			&tag,
		); err != nil {
			return nil, err
		}

		post, exists := postsMap[id]
		if !exists {
			post = &model.AuthorPost{
				Post: model.Post{
					ID: id,
					AuthorID: authorID,
					Title: title,
					Content: content,
					Views: views,
					Likes: likes,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Images: []*model.PostImage{},
				Tags: []string{},
			}
			postsMap[post.Post.ID] = post
		}

		if imageURL != nil && imagePosition != nil {
			postsMap[post.Post.ID].Images = append(postsMap[post.Post.ID].Images, &model.PostImage{URL: *imageURL, Position: *imagePosition})
		}

		if tag != nil {
			postsMap[post.Post.ID].Tags = append(postsMap[post.Post.ID].Tags, *tag)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var posts []*model.AuthorPost
	for _, post := range postsMap {
		posts = append(posts, post)
	}

	return posts, nil
}

func (r *postRepo) SearchByTags(ctx context.Context, tags []string, limit int, offset int) ([]*model.FullPost, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at, t.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE t.tag = ANY($1)
		LIMIT $2
		OFFSET $3
		ORDER BY p.likes DESC
		ORDER BY p.views DESC
		ORDER BY p.created_at DESC`,
		tags,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postsMap := make(map[int64]*model.FullPost)
	for rows.Next() {
		var (
			id int64
			authorID uuid.UUID
			title string
			content string
			views int64
			likes int64
			createdAt time.Time
			updatedAt time.Time
			username string
			displayName *string
			avatarURL *string
			imageURL *string
			imagePosition *int
			tag *string
		)
		if err := rows.Scan(
			&id,
			&authorID,
			&title,
			&content,
			&views,
			&likes,
			&createdAt,
			&updatedAt,
			&username,
			&displayName,
			&avatarURL,
			&imageURL,
			&imagePosition,
			&tag,
		); err != nil {
			return nil, err
		}

		post, exists := postsMap[id]
		if !exists {
			post = &model.FullPost{
				Post: model.Post{
					ID: id,
					AuthorID: authorID,
					Title: title,
					Content: content,
					Views: views,
					Likes: likes,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Author: model.UserAuthor{
					Username: username,
					DisplayName: displayName,
					AvatarURL: avatarURL,
				},
				Images: []*model.PostImage{},
				Tags: []string{},
			}
			postsMap[id] = post
		}

		if imageURL != nil && imagePosition != nil {
			postsMap[post.Post.ID].Images = append(postsMap[post.Post.ID].Images, &model.PostImage{URL: *imageURL, Position: *imagePosition})
		}

		if tag != nil {
			postsMap[post.Post.ID].Tags = append(postsMap[post.Post.ID].Tags, *tag)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var posts []*model.FullPost
	for _, post := range postsMap {
		posts = append(posts, post)
	}

	if len(posts) == 0 {
		return nil, pgx.ErrNoRows
	}

	return posts, nil
}

func (r *postRepo) IncrViews(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, "UPDATE posts SET views = views + 1 WHERE id = $1", id)
	return err
}
