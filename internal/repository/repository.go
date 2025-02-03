package repository

import (
	"github.com/BloggingApp/post-service/internal/repository/postgres"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Repository struct {
	Postgres *postgres.PostgresRepository
	Redis    *redisrepo.RedisRepository
}

func New(db *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *Repository {
	return &Repository{
		Postgres: postgres.New(db, logger),
		Redis: redisrepo.New(rdb),
	}
}
