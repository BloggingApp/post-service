package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/redisrepo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type postService struct {
	logger *zap.Logger
	repo *repository.Repository
}

func newPostService(logger *zap.Logger, repo *repository.Repository) Post {
	return &postService{
		logger: logger,
		repo: repo,
	}
}

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

		buff := make([]byte, 512)
		if _, err := file.Read(buff); err != nil {
			s.logger.Sugar().Errorf("error while creating a post for user(%s): %s", authorID.String())
			return nil, ErrInternal
		}
	
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			s.logger.Sugar().Errorf("error while creating a post for user(%s): %s", authorID.String(), err.Error())
			return nil, ErrInternal
		}
	
		if !strings.HasPrefix(http.DetectContentType(buff), "image/") {
			return nil, ErrFileMustBeImage
		}

		ext := filepath.Ext(img.FileHeader.Filename)
		if ext == "" {
			return nil, ErrFileMustHaveAValidExtension
		}

		imgID := uuid.New()
		filePath := "public/post-images/" + imgID.String() + ext
		if _,err := os.Create(filePath); err != nil {
			s.logger.Sugar().Errorf("failed to create file for author(%s) post image: %s", authorID.String(), err.Error())
			return nil, ErrInternal
		}

		imgURL := fmt.Sprintf("%s/%s", viper.GetString("app.url"), filePath)
		images = append(images, &model.PostImage{URL: imgURL, Position: img.Position})
	}

	createdPost, err := s.repo.Postgres.Post.Create(ctx, post, images, dto.Tags)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user(%s) post: %s", post.AuthorID.String(), err.Error())
		return nil, ErrInternal
	}

	return createdPost, nil
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

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.PostKey(id), post, time.Hour); err != nil {
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
