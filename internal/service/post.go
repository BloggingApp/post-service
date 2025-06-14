package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/rabbitmq"
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
	rabbitmq *rabbitmq.MQConn
}

func newPostService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) Post {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	return &postService{
		logger: logger,
		repo: repo,
		httpClient: &http.Client{},
		scheduler: scheduler,
		rabbitmq: rabbitmq,
	}
}

const (
	POST_LIKES_UPDATE_TIMEOUT = time.Minute * 2
	COMMENT_LIKES_UPDATE_TIMEOUT = time.Minute * 2
)

var REGEXP_TO_GET_IMAGES = regexp.MustCompile(`!\[.*?\]\((.*?)\)`)

func (s *postService) UploadTempPostImage(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	filePath := "/post-images/temp/" + uuid.NewString() + uuid.NewString() + "." + filepath.Ext(fileHeader.Filename)
	return s.uploadImageToFileStorage(filePath, file, fileHeader)
}

func (s *postService) Create(ctx context.Context, authorID uuid.UUID, req dto.CreatePostRequest) (*model.Post, error) {
	post := model.Post{
		AuthorID: authorID,
		Title: req.Title,
		Content: req.Content,
	}

	createdPost, err := s.repo.Postgres.Post.Create(ctx, post, req.Tags)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user(%s) post: %s", post.AuthorID.String(), err.Error())
		return nil, ErrInternal
	}

	matches := REGEXP_TO_GET_IMAGES.FindAllStringSubmatch(post.Content, -1)

	moves := make(map[string]string)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		url := match[1]

		if strings.Contains(url, "/temp/") {
			oldPath := s.extractPathFromURL(url)
			newPath := strings.Replace(oldPath, "/temp/", "/perm/", 1)

			moves[oldPath] = newPath

			newURL := strings.Replace(url, "/temp", "/perm/", 1)
			post.Content = strings.ReplaceAll(post.Content, url, newURL)
		}
	}

	if err := s.moveImagesFromTempToPerm(moves); err != nil {
		s.logger.Sugar().Errorf("failed to move user(%s)'s post images from temp to perm: %s", authorID.String(), err.Error())
		return nil, ErrInternal
	}
	
	postCreatedMsg := dto.MQPostCreatedMsg{
		PostID: createdPost.ID,
		UserID: authorID,
		PostTitle: createdPost.Title,
		CreatedAt: createdPost.CreatedAt,
	}
	postCreatedMsgJSON, err := json.Marshal(postCreatedMsg)
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal user(%s)'s post created msg to json: %s", authorID.String(), err.Error())
		return nil, ErrInternal
	}
	if err := s.rabbitmq.PublishToQueue(rabbitmq.NEW_POST_NOTIFICATION_QUEUE, postCreatedMsgJSON); err != nil {
		s.logger.Sugar().Errorf("failed to publish user(%s)'s new post notification to rabbitmq: %s", authorID.String(), err.Error())
		return nil, ErrInternal
	}

	return createdPost, nil
}

func (s *postService) extractPathFromURL(url string) string {
	u, err := urlpkg.Parse(url)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(u.Path, "/") {
		u.Path = u.Path[:len(u.Path)-1]
	}
	return u.Path
}

func (s *postService) moveImagesFromTempToPerm(moves map[string]string) error {
	jsonBody, _ := json.Marshal(moves)

	req, err := http.NewRequest(http.MethodPost, viper.GetString("file-storage.origin") + "/move", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to move files")
	}

	return nil
}

func (s *postService) uploadImageToFileStorage(path string, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	endpoint := "/upload"
	url := viper.GetString("file-storage.origin") + endpoint

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Writing text fields
	if err := writer.WriteField("type", "IMAGE"); err != nil {
		s.logger.Sugar().Errorf("failed to write 'type' field for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if err := writer.WriteField("path", path); err != nil {
		s.logger.Sugar().Errorf("failed to write 'path' field for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	// Writing file
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

	// End of request body
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

func (s *postService) GetTrending(ctx context.Context, hours, limit int) ([]*model.FullPost, error) {
	if hours > 24 * 7 {
		hours = 24 * 7
	}

	postsCache, err := redisrepo.GetMany[model.FullPost](s.repo.Redis.Default, ctx, redisrepo.TrendingPostsKey(limit))
	if err == nil {
		return postsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get trending posts with limit(%d) from redis: %s", limit, err.Error())
		return nil, ErrInternal
	}

	posts, err := s.repo.Postgres.Post.GetTrending(ctx, hours, limit)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get trending posts with limit(%d) from postgres: %s", limit, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.TrendingPostsKey(limit), posts, time.Duration(hours * int(time.Hour))); err != nil {
		s.logger.Sugar().Errorf("failed to set trending posts for limit(%d) in redis cache: %s", limit, err.Error())
		return nil, ErrInternal
	}

	return posts, nil
}

func (s *postService) SearchByTitle(ctx context.Context, title string, limit, offset int) ([]*model.FullPost, error) {
	resultCache, err := redisrepo.GetMany[model.FullPost](s.repo.Redis.Default, ctx, redisrepo.SearchPostsResultByTitleKey(title, limit, offset))
	if err == nil {
		return resultCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get posts search result by title(%s) from redis: %s", title, err.Error())
		return nil, ErrInternal
	}

	result, err := s.repo.Postgres.Post.SearchByTitle(ctx, title, limit, offset)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get posts search result by title(%s) from postgres: %s", title, err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.SearchPostsResultByTitleKey(title, limit, offset), result, time.Minute); err != nil {
		s.logger.Sugar().Errorf("failed to set posts search result by title(%s) in redis: %s", title, err.Error())
		return nil, ErrInternal
	}

	return result, nil
}

func (s *postService) Edit(ctx context.Context, dto dto.EditPostRequest) error {
	post, err := s.repo.Postgres.Post.FindByID(ctx, dto.ID)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get post(%d) from postres: %s", dto.ID, err.Error())
		return ErrInternal
	}

	updates := make(map[string]any)

	if dto.Content != nil {
		editedContent := *dto.Content

		newUrls := []string{}
		oldUrls := []string{}

		matches := REGEXP_TO_GET_IMAGES.FindAllStringSubmatch(editedContent, -1)

		moves := make(map[string]string)

		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			url := match[1]
			newUrls = append(newUrls, url)

			if strings.Contains(url, "/temp/") {
				oldPath := s.extractPathFromURL(url)
				newPath := strings.Replace(oldPath, "/temp/", "/perm/", 1)

				moves[oldPath] = newPath

				newURL := strings.Replace(url, "/temp", "/perm/", 1)
				editedContent = strings.ReplaceAll(editedContent, url, newURL)
			}
		}

		if err := s.moveImagesFromTempToPerm(moves); err != nil {
			s.logger.Sugar().Errorf("failed to move user(%s)'s post images from temp to perm: %s", post.Post.AuthorID.String(), err.Error())
			return ErrInternal
		}

		matches = REGEXP_TO_GET_IMAGES.FindAllStringSubmatch(post.Post.Content, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			url := match[1]
			oldUrls = append(oldUrls, url)
		}

		for i, oldUrl := range oldUrls {
			for _, newUrl := range newUrls {
				if oldUrl == newUrl {
					oldUrls = append(oldUrls[:i], oldUrl[i+1:])
				}
			}
		}

		removedPaths := []string{}
		for _, url := range oldUrls {
			removedPaths = append(removedPaths, s.extractPathFromURL(url))
		}

		if err := s.deletePostImages(removedPaths); err != nil {
			s.logger.Sugar().Errorf("failed to delete post(%d) removed urls: %s", post.Post.ID, err.Error())
			return ErrInternal
		}

		updates["content"] = editedContent
	}

	if dto.Title != nil {
		updates["title"] = *dto.Title
	}

	if err := s.repo.Postgres.Post.UpdateByID(ctx, post.Post.ID, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update post(%d): %s", post.Post.ID, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *postService) deletePostImages(paths []string) error {
	jsonBody, _ := json.Marshal(paths)

	req, err := http.NewRequest(http.MethodPost, viper.GetString("file-storage.origin") + "/delete", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete files")
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
