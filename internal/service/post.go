package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type postService struct {
	logger *zap.Logger
	repo *repository.Repository
	httpClient *http.Client
	scheduler gocron.Scheduler
}

func newPostService(logger *zap.Logger, repo *repository.Repository) Post {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	return &postService{
		logger: logger,
		repo: repo,
		httpClient: &http.Client{},
		scheduler: scheduler,
	}
}

const (
	POST_LIKES_UPDATE_TIMEOUT = time.Minute * 2
	COMMENT_LIKES_UPDATE_TIMEOUT = time.Minute * 2
)

func (s *postService) Create(ctx context.Context, authorID uuid.UUID, dto dto.CreatePostDto, imagesDto []dto.CreatePostImagesDto) (*model.Post, error) {
	post := model.Post{
		AuthorID: authorID,
		Title: dto.Title,
		Content: dto.Content,
	}

	var images []*model.PostImage
	for _, img := range imagesDto {
		file, err := img.FileHeader.Open()
		if err != nil {
			s.logger.Sugar().Errorf("failed to open file: %s", err.Error())
			return nil, ErrInternal
		}
		defer file.Close()

		uploadPath := "post-images"

		returnedURL, err := s.uploadImageToCDN(uploadPath, file, img.FileHeader)
		if err != nil {
			return nil, err
		}

		images = append(images, &model.PostImage{URL: returnedURL, Position: img.Position})
	}

	createdPost, err := s.repo.Postgres.Post.Create(ctx, post, images, dto.Tags)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user(%s) post: %s", post.AuthorID.String(), err.Error())
		return nil, ErrInternal
	}

	return createdPost, nil
}

func (s *postService) uploadImageToCDN(path string, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	endpoint := "/upload"
	url := viper.GetString("cdn.origin") + endpoint

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fileWriter, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create file part for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		s.logger.Sugar().Errorf("failed to seek to the start of the file: %s", err.Error())
		return "", ErrInternal
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		s.logger.Sugar().Errorf("failed to copy file content for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if err := writer.WriteField("path", path); err != nil {
		s.logger.Sugar().Errorf("failed to write path field for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if err := writer.Close(); err != nil {
		s.logger.Sugar().Errorf("failed to close writer for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	req, err := http.NewRequest(http.MethodPost, url, &requestBody)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create CDN request: %s", err.Error())
		return "", ErrInternal
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("type", "IMAGE")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Sugar().Errorf("failed to do CDN request: %s", err.Error())
		return "", ErrInternal
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Sugar().Errorf("failed to read response body from CDN: %s", err.Error())
		return "", ErrInternal
	}

	if resp.StatusCode != http.StatusOK {
		var bodyJSON map[string]interface{}
        if err := json.Unmarshal(body, &bodyJSON); err != nil {
            s.logger.Sugar().Errorf("failed to decode error response from CDN: %s", err.Error())
        } else {
            s.logger.Sugar().Errorf("ERROR from CDN endpoint(%s), code(%d), details: %s", endpoint, resp.StatusCode, bodyJSON["details"])
        }
        return "", ErrFailedToUploadPostImageToCDN
	}

	return string(body), nil
}

func (s *postService) FindByID(ctx context.Context, id int64) (*model.FullPost, error) {
	cachedPost, err := redisrepo.Get[model.FullPost](s.repo.Redis.Default, ctx, redisrepo.PostKey(id))
	if err == nil {
		if cachedPost != nil {
			s.incrViewsIfPostIsNotNil(cachedPost.Post.ID)
		}
		return cachedPost, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get post(%d) from redis: %s", id, err.Error())
		return nil, err
	}

	post, err := s.repo.Postgres.Post.FindByID(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Sugar().Errorf("failed to find post(%d) from postgres: %s", id, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.PostKey(id), post, time.Minute * 30); err != nil {
		s.logger.Sugar().Errorf("failed to set post(%d) in redis: %s", id, err.Error())
		return nil, ErrInternal
	}

	if post != nil {
		go s.incrViewsIfPostIsNotNil(post.Post.ID)
	}

	return post, nil
}

func (s *postService) incrViewsIfPostIsNotNil(postID int64) {
	go func(id int64) {
		ctx := context.Background()
		if err := s.repo.Postgres.Post.IncrViews(ctx, id); err != nil {
			s.logger.Sugar().Errorf("failed to increment views for post(%d): %s", id, err.Error())
		}
	}(postID)
}

func (s *postService) FindAuthorPosts(ctx context.Context, authorID uuid.UUID, limit int, offset int) ([]*model.AuthorPost, error) {
	maxLimit(&limit)

	cachedPosts, err := redisrepo.GetMany[model.AuthorPost](s.repo.Redis.Default, ctx, redisrepo.AuthorPostsKey(authorID.String(), limit, offset))
	if err == nil {
		return cachedPosts, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get author(%s) posts from redis: %s", authorID.String(), err.Error())
		return nil, ErrInternal
	}

	posts, err := s.repo.Postgres.Post.FindAuthorPosts(ctx, authorID, limit, offset)
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Sugar().Errorf("failed to find author(%s) posts from postgres: %s", authorID.String(), err.Error())
		return nil, err
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.AuthorPostsKey(authorID.String(), limit, offset), posts, time.Hour); err != nil {
		s.logger.Sugar().Errorf("failed to set author(%s) posts in redis: %s", authorID.String(), err.Error())
		return nil, err
	}

	return posts, nil
}

func (s *postService) FindUserLikes(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]*model.FullPost, error) {
	maxLimit(&limit)

	postsCache, err := redisrepo.GetMany[model.FullPost](s.repo.Redis.Default, ctx, redisrepo.UserLikesKey(userID.String(), limit, offset))
	if err == nil {
		return postsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get user(%s) likes from redis: %s", userID.String(), err.Error())
		return nil, ErrInternal
	}

	posts, err := s.repo.Postgres.Post.FindUserLikes(ctx, userID, limit, offset)
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Sugar().Errorf("failed to get user(%s) likes from postgres: %s", userID.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserLikesKey(userID.String(), limit, offset), posts, time.Hour); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) likes in redis: %s", userID.String(), err.Error())
		return nil, ErrInternal
	}

	return posts, nil
}

func (s *postService) IsLiked(ctx context.Context, postID int64, userID uuid.UUID) bool {
	isLikedCache, err := s.repo.Redis.Default.Get(ctx, redisrepo.IsLikedPostKey(userID.String(), postID)).Bool()
	if err == nil {
		return isLikedCache
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get if user(%s) is liked post(%d) from redis: %s", userID.String(), postID, err.Error())
		return false
	}

	isLiked := s.repo.Postgres.Post.IsLiked(ctx, postID, userID)

	if err := s.repo.Redis.Default.Set(ctx, redisrepo.IsLikedPostKey(userID.String(), postID), isLiked, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to set if user(%s) is liked post(%d) in redis: %s", userID.String(), postID, err.Error())
		return false
	}

	return isLiked
}

// Set 'unlike' value to true if you want to UNLIKE a post
func (s *postService) Like(ctx context.Context, postID int64, userID uuid.UUID, unlike bool) error {
	var affected bool
	var delta int64
	if unlike {
		affected = s.repo.Postgres.Post.Unlike(ctx, postID, userID)
		delta = -1
	} else {
		affected = s.repo.Postgres.Post.Like(ctx, postID, userID)
		delta = 1
	}

	if !affected {
		return ErrFailedToLikeThePost
	}

	// Update "is liked" cache
	if err := s.repo.Redis.Default.Set(ctx, redisrepo.IsLikedPostKey(userID.String(), postID), !unlike, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to delete user(%s) is liked for post(%d) from redis: %s", userID.String(), postID, err.Error())
		return ErrInternal
	}
	
	if err := s.updatePostCachedLikes(ctx, postID, delta); err != nil {
		return err
	}

	return nil
}

func (s *postService) updatePostCachedLikes(ctx context.Context, postID int64, delta int64) error {
	likesKey := redisrepo.PostLikesKey(postID)

	if err := s.repo.Redis.Default.IncrBy(ctx, likesKey, delta).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to increment key(%s) in redis: %s", likesKey, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *postService) postsBatchLikesUpdate(ctx context.Context) error {
	postKeys, err := s.repo.Redis.Default.Keys(ctx, redisrepo.POST_LIKES_KEY_PATTERN).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get keys with pattern(%s) from redis: %s", redisrepo.POST_LIKES_KEY_PATTERN, err.Error())
	}
	if err == redis.Nil || len(postKeys) == 0 {
		return nil
	}

	for _, postKey := range postKeys {
		// Getting postID from redis key
		postID, err := redisrepo.GetPostIDFromPostLikesKey(postKey)
		if err != nil {
			continue
		}

		// Getting post's cached likes
		n, err := s.repo.Redis.Default.Get(ctx, postKey).Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to get post(%d) cached likes from redis: %s", postID, err.Error())
		}
		if err == redis.Nil {
			continue
		}

		if err := s.repo.Postgres.Post.IncrPostLikesBy(ctx, postID, n); err != nil {
			return fmt.Errorf("failed to incr post(%d) likes by(%d): %s", postID, n, err.Error())
		}

		if err := s.repo.Redis.Default.Del(ctx, postKey).Err(); err != nil {
			return fmt.Errorf("failed to delete post(%d) likes from redis: %s", postID, err.Error())
		}
	}

	return nil
}

func (s *postService) SchedulePostLikesUpdates() {
	s.scheduler.NewJob(gocron.DurationJob(POST_LIKES_UPDATE_TIMEOUT), gocron.NewTask(func(ctx context.Context) {
		if err := s.postsBatchLikesUpdate(ctx); err != nil {
			s.logger.Sugar().Error(err.Error())
		}
	}))
}

func (s *postService) StartScheduledJobs() {
	s.SchedulePostLikesUpdates()

	s.scheduler.Start()
}
