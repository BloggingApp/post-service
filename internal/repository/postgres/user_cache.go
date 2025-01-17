package postgres

import (
	"context"
	"strconv"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userCacheRepo struct {
	db *pgx.Conn
}

func newUserCacheRepo(db *pgx.Conn) UserCache {
	return &userCacheRepo{
		db: db,
	}
}

func (r *userCacheRepo) Create(ctx context.Context, cachedUser model.CachedUser) error {
	_, err := r.db.Exec(ctx, "INSERT INTO cached_users(id, username, display_name, avatar_url) VALUES($1, $2, $3, $4)", cachedUser.ID, cachedUser.Username, cachedUser.DisplayName, cachedUser.AvatarURL)
	return err
}

func (r *userCacheRepo) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	allowedFields := []string{"username", "display_name", "avatar_url"}
	allowedFieldsSet := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowedFieldsSet[field] = struct{}{}
	}

	for field := range updates {
		if _, ok := allowedFieldsSet[field]; !ok {
			return ErrFieldsNotAllowedToUpdate
		}
	}

	query := "UPDATE cached_users SET "
	args := []interface{}{}
	i := 1

	for column, value := range updates {
		query += (column + " = $" + strconv.Itoa(i) + ", ")
		args = append(args, value)
		i++
	}

	query = query[:len(query)-2] + " WHERE id = $" + strconv.Itoa(i)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *userCacheRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.CachedUser, error) {
	var user model.CachedUser
	if err := r.db.QueryRow(
		ctx,
		"SELECT u.id, u.username, u.display_name, u.avatar_url FROM cached_users u WHERE u.id = $1",
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.AvatarURL,
	); err != nil {
		return nil, err
	}

	return &user, nil
}
