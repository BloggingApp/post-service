package repository

import (
	"github.com/BloggingApp/post-service/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repository struct {
	Postgres *postgres.PostgresRepository
}

func New(db *pgxpool.Pool, logger *zap.Logger) *Repository {
	return &Repository{
		Postgres: postgres.New(db, logger),
	}
}
