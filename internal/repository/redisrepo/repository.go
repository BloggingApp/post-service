package redisrepo

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Default interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Decr(ctx context.Context, key string) *redis.IntCmd
	IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
}

type RedisRepository struct {
	Default
}

func New(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{
		Default: newDefaultRepo(rdb),
	}
}
