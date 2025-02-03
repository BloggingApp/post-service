package redisrepo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type defaultRepo struct {
	rdb *redis.Client
}

func newDefaultRepo(rdb *redis.Client) Default {
	return &defaultRepo{
		rdb: rdb,
	}
}

func (r *defaultRepo) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.rdb.Set(ctx, key, value, ttl).Err()
}

func (r *defaultRepo) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.rdb.Set(ctx, key, valueJSON, ttl).Err()
}

func (r *defaultRepo) Get(ctx context.Context, key string) *redis.StringCmd {
	return r.rdb.Get(ctx, key)
}

func Get[T any](r Default, ctx context.Context, key string) (*T, error) {
	value, err := r.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if value == "null" {
		return nil, nil
	}

	var result T
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func GetMany[T any](r Default, ctx context.Context, key string) ([]*T, error) {
	value, err := r.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if value == "null" {
		return nil, nil
	}

	var result []*T
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *defaultRepo) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.rdb.Del(ctx, keys...)
}

func (r *defaultRepo) Incr(ctx context.Context, key string) *redis.IntCmd {
	return r.rdb.Incr(ctx, key)
}

func (r *defaultRepo) Decr(ctx context.Context, key string) *redis.IntCmd {
	return r.rdb.Decr(ctx, key)
}

func (r *defaultRepo) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return r.rdb.IncrBy(ctx, key, value)
}

func (r *defaultRepo) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return r.rdb.DecrBy(ctx, key, value)
}

func (r *defaultRepo) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return r.rdb.Keys(ctx, pattern)
}
