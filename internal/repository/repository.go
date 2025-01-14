package repository

import (
	"github.com/BloggingApp/post-service/internal/repository/postgres"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	Postgres *postgres.PostgresRepository
	Redis    *redisrepo.RedisRepository
}

func New(db *pgx.Conn, rdb *redis.Client) *Repository {
	return &Repository{
		Postgres: postgres.New(db),
		Redis: redisrepo.New(rdb),
	}
}
