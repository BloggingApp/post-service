package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/rabbitmq"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type userCacheService struct {
	logger *zap.Logger
	repo *repository.Repository
	rabbitmq *rabbitmq.MQConn
	httpClient *http.Client
}

func newUserCacheService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) UserCache {
	return &userCacheService{
		logger: logger,
		repo: repo,
		rabbitmq: rabbitmq,
		httpClient: &http.Client{},
	}
}

func (s *userCacheService) CreateOrGet(ctx context.Context, id uuid.UUID, accessToken string) (*model.CachedUser, error) {
	cachedUser, err := s.FindByID(ctx, id)
	if err == nil {
		return cachedUser, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	fetchedUser, err := s.fetchUser(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Postgres.UserCache.Create(ctx, *fetchedUser); err != nil {
		s.logger.Sugar().Errorf("failed to create cached user(%s): %s", fetchedUser.ID.String(), err.Error())
		return nil, ErrInternal
	}

	return fetchedUser, nil
}

func (s *userCacheService) fetchUser(ctx context.Context, accessToken string) (*model.CachedUser, error) {
    endpoint := "/users/@me"
    url := viper.GetString("user-service.api") + endpoint

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        s.logger.Sugar().Errorf("failed to create request to user-service: %s", err.Error())
        return nil, ErrInternal
    }

    req.Header.Add("Authorization", "Bearer "+accessToken)

    resp, err := s.httpClient.Do(req)
    if err != nil {
        s.logger.Sugar().Errorf("failed to send request to user-service: %s", err.Error())
        return nil, ErrInternal
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        s.logger.Sugar().Errorf("failed to read response body from user-service: %s", err.Error())
        return nil, ErrInternal
    }

    if resp.StatusCode != http.StatusOK {
        var bodyJSON map[string]interface{}
        if err := json.Unmarshal(body, &bodyJSON); err != nil {
            s.logger.Sugar().Errorf("failed to decode error response from user-service: %s", err.Error())
        } else {
            s.logger.Sugar().Errorf("ERROR from user-service endpoint(%s), details: %s", endpoint, bodyJSON["details"])
        }
        return nil, errors.New("failed to fetch user")
    }
	
    var user model.CachedUser
    if err := json.Unmarshal(body, &user); err != nil {
        s.logger.Sugar().Errorf("failed to decode user response body from user-service: %s", err.Error())
        return nil, ErrInternal
    }

    return &user, nil
}

func (s *userCacheService) Create(ctx context.Context, cachedUser model.CachedUser) error {
	if err := s.repo.Postgres.UserCache.Create(ctx, cachedUser); err != nil {
		s.logger.Sugar().Errorf("failed to create cached user(%s): %s", cachedUser.ID.String(), err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userCacheService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := s.repo.Postgres.UserCache.Update(ctx, id, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update cached user(%s): %s", id.String(), err.Error())
		return ErrInternal
	}

	if err := s.repo.Redis.Default.Del(ctx, redisrepo.UserCacheKey(id.String())).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete cached user(%s) from redis: %s", id.String(), err.Error())
	}

	return nil
}

func (s *userCacheService) FindByID(ctx context.Context, id uuid.UUID) (*model.CachedUser, error) {
	cachedUser, err := redisrepo.Get[model.CachedUser](s.repo.Redis.Default, ctx, redisrepo.UserCacheKey(id.String()))
	if err == nil {
		return cachedUser, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get cached user(%s) from redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	user, err := s.repo.Postgres.UserCache.FindByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}

		s.logger.Sugar().Errorf("failed to get cached user(%s) from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserCacheKey(id.String()), user, time.Hour); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	return user, nil
}

func (s *userCacheService) consumeUserUpdates(ctx context.Context) {
	queue := rabbitmq.USER_INFO_UPDATED_QUEUE
	msgs, err := s.rabbitmq.Consume(queue)
	if err != nil {
		s.logger.Sugar().Fatalf("failed to start consume updates from queue(%s): %s", queue, err.Error())
	}

	for msg := range msgs {
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Body, &data); err != nil {
			s.logger.Sugar().Errorf("failed to unmarshal json in queue(%s): %s", queue, err.Error())
			msg.Nack(false, false)
			continue
		}

		userIDString, exists := data["user_id"].(string)
		if !exists {
			s.logger.Sugar().Errorf("'user_id' field is not provided")
			msg.Nack(false, false)
			continue
		}
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			s.logger.Sugar().Errorf("provided an invalid user_id")
			msg.Nack(false, false)
			continue
		}

		delete(data, "user_id")

		if err := s.Update(ctx, userID, data); err != nil {
			msg.Nack(false, true)
			continue
		}

		msg.Ack(false)
	}
}
