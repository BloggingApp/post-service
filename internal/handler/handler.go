package handler

import (
	"context"
	"os"

	"github.com/BloggingApp/post-service/internal/model"
	"github.com/BloggingApp/post-service/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
	"github.com/spf13/viper"
)

type Handler struct {
	services *service.Service
}

func New(services *service.Service) *Handler {
	return &Handler{
		services: services,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{viper.GetString("client.origin")},
		AllowMethods: []string{"POST", "GET"},
		AllowCredentials: true,
	}))
	
	v1 := r.Group("/api/v1")
	{
		posts := v1.Group("/posts")
		{
			posts.POST("", h.authMiddleware, h.postsCreate)
			posts.GET("/my", h.authMiddleware, h.postsGetMy)
			posts.GET("/author/:userID", h.postsGet)
			posts.GET("/liked", h.authMiddleware, h.postsGetLiked)
			posts.GET("/trending", h.authMiddleware, h.postsTrending)

			post := posts.Group("/:postID")
			{
				post.GET("", h.notRequiredAuthMiddleware, h.postsGetByID)
				post.POST("/like", h.authMiddleware, h.postsLike)
				post.DELETE("/unlike", h.authMiddleware, h.postsUnlike)
				post.GET("/isLiked", h.authMiddleware, h.postsIsLiked)
			}
		}

		comments := v1.Group("/comments")
		{
			comments.POST("", h.authMiddleware, h.commentsCreate)

			postComments := comments.Group("/:postID")
			{
				postComments.GET("", h.commentsGet)

				comment := postComments.Group("/:commentID")
				{
					comment.GET("/replies", h.commentsGetReplies)
					comment.DELETE("", h.authMiddleware, h.commentsDelete)
					comment.GET("/isLiked", h.authMiddleware, h.commentsIsLiked)
					comment.POST("/like", h.authMiddleware, h.commentsLike)
					comment.DELETE("/unlike", h.authMiddleware, h.commentsUnlike)
				}
			}
		}
	}

	return r
}

func (h *Handler) getUserDataFromAccessTokenClaims(ctx context.Context, accessToken string) (*model.CachedUser, error) {
	claims, err := jwtmanager.DecodeJWT(accessToken, []byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}

	idString := claims["id"].(string)
	id, err := uuid.Parse(idString)
	if err != nil {
		return nil, err
	}

	user, err := h.services.UserCache.CreateOrGet(ctx, id, accessToken)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *Handler) getCachedUserFromRequest(c *gin.Context) *model.CachedUser {
	userReq, _ := c.Get("cached-user")

	user, ok := userReq.(model.CachedUser)
	if !ok {
		return nil
	}

	return &user
}
