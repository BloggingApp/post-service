package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type postRepo struct {
	db *pgxpool.Pool
	logger *zap.Logger
}

func newPostRepo(db *pgxpool.Pool, logger *zap.Logger) Post {
	return &postRepo{
		db: db,
		logger: logger,
	}
}

func (r *postRepo) Create(ctx context.Context, post model.Post, tags []string) (*model.Post, error) {
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
		return nil, nil
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
		ORDER BY p.created_at DESC
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
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at, u.username, u.display_name, u.avatar_url, i.url, i.position, t.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE t.tag = ANY($1)
		ORDER BY p.likes DESC, p.views DESC, p.created_at DESC
		LIMIT $2
		OFFSET $3`,
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
		return []*model.FullPost{}, nil
	}

	return posts, nil
}

func (r *postRepo) IncrViews(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, "UPDATE posts SET views = views + 1 WHERE id = $1", id)
	return err
}

func (r *postRepo) Like(ctx context.Context, postID int64, userID uuid.UUID) bool {
	cmd, err := r.db.Exec(ctx, "INSERT INTO post_likes(post_id, user_id) VALUES($1, $2) ON CONFLICT DO NOTHING", postID, userID)
	return err == nil && cmd.RowsAffected() == 1
}

func (r *postRepo) IncrPostLikesBy(ctx context.Context, postID int64, n int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE posts
		SET likes = CASE
			WHEN likes + $1 >= 0 THEN likes + $1
			ELSE 0
		END
		WHERE id = $2
	`, n, postID)
	return err
}

func (r *postRepo) Unlike(ctx context.Context, postID int64, userID uuid.UUID) bool {
	cmd, err := r.db.Exec(ctx, "DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2", postID, userID)
	return err == nil && cmd.RowsAffected() == 1
}

func (r *postRepo) IsLiked(ctx context.Context, postID int64, userID uuid.UUID) bool {
	var exists bool
	var err error

	for retries := 0; retries < 3; retries++ {
		err = r.db.QueryRow(ctx, "SELECT count(*) > 0 FROM post_likes WHERE post_id = $1 AND user_id = $2", postID, userID).Scan(&exists)
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

func (r *postRepo) FindUserLikes(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]*model.FullPost, error) {
	maxLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at, u.username, u.display_name, u.avatar_url, i.url, i.position, t.tag
		FROM post_likes l
		JOIN posts p ON l.post_id = p.id
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE l.user_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2
		OFFSET $3`,
		userID,
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
		return []*model.FullPost{}, nil
	}

	return posts, nil
}

func (r *postRepo) GetTrending(ctx context.Context, hours, limit int) ([]*model.FullPost, error) {
	maxLimit(&limit)

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at,
		u.username, u.display_name, u.avatar_url,
		i.url, i.position,
		t.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE p.created_at >= $1
		ORDER BY p.likes DESC, p.views DESC
		LIMIT $2
		`,
		since, limit,
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
		return []*model.FullPost{}, nil
	}

	return posts, nil
}

func (r *postRepo) SearchByTitle(ctx context.Context, title string, limit, offset int) ([]*model.FullPost, error) {
	maxLimit(&limit)

	search := "%" + title + "%"

	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		p.id, p.author_id, p.title, p.content, p.views, p.likes, p.created_at, p.updated_at,
		u.username, u.display_name, u.avatar_url,
		i.url, i.position,
		t.tag
		FROM posts p
		JOIN cached_users u ON p.author_id = u.id
		LEFT JOIN post_images i ON p.id = i.post_id
		LEFT JOIN post_tags t ON p.id = t.post_id
		WHERE p.title LIKE $1
		ORDER BY p.created_at DESC, p.updated_at DESC
		LIMIT $2
		OFFSET $3
		`, search, limit, offset,
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
		return []*model.FullPost{}, nil
	}

	return posts, nil
}

func (r *postRepo) Update(ctx context.Context, id int64, authorID uuid.UUID, fields map[string]any) error {
	allowedFields := []string{"title", "content"}
	updates := map[string]any{}
	for _, allowedField := range allowedFields {
		for field, value := range fields {
			if field == allowedField {
				updates[field] = value
			}
		}
	}

	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE posts SET "
	args := []interface{}{}
	i := 1

	for column, value := range updates {
		query += (column + " = $" + strconv.Itoa(i) + ", ")
		args = append(args, value)
		i += 1
	}

	query = query[:len(query)-2] + " WHERE id = $" + strconv.Itoa(i) + " AND author_id = $" + strconv.Itoa(i+1)
	args = append(args, id, authorID)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}
