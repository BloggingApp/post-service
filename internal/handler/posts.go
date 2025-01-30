package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) postsCreate(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.CreatePostDto
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	var images []dto.CreatePostImagesDto
	for positionString, files := range form.File {
		positionInt, err := strconv.Atoi(positionString)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errPositionMustBeInt.Error()))
			return
		}

		for _, file := range files {
			images = append(images, dto.CreatePostImagesDto{
				Position: positionInt,
				FileHeader: file,
			})
		}
	}

	createdPost, err := h.services.Post.Create(c.Request.Context(), user.ID, input, images)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, *createdPost)
}

func (h *Handler) postsGetMy(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.GetPostsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	posts, err := h.services.Post.FindAuthorPosts(c.Request.Context(), user.ID, input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *Handler) postsGetByID(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	postIDString := strings.TrimSpace(c.Param("postID"))
	postID, err := strconv.Atoi(postIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidPostID.Error()))
		return
	}

	post, err := h.services.Post.FindByID(c.Request.Context(), int64(postID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	postDto := dto.GetPost{
		Post: *post,
	}

	if user != nil {
		isLiked := h.services.Post.IsLiked(c.Request.Context(), post.Post.ID, user.ID)
		postDto.IsLiked = isLiked
	}

	c.JSON(http.StatusOK, postDto)
}

func (h *Handler) postsGet(c *gin.Context) {
	var input dto.GetPostsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	userIDString := strings.TrimSpace(c.Param("userID"))
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	posts, err := h.services.Post.FindAuthorPosts(c.Request.Context(), userID, input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *Handler) postsIsLiked(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	postIDString := c.Param("postID")
	postID, err := strconv.Atoi(postIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	isLiked := h.services.Post.IsLiked(c.Request.Context(), int64(postID), user.ID)

	c.JSON(http.StatusOK, gin.H{"isLiked": isLiked})
}

func (h *Handler) postsLike(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	postIDString := c.Param("postID")
	postID, err := strconv.Atoi(postIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Post.Like(c.Request.Context(), int64(postID), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) postsUnlike(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	postIDString := c.Param("postID")
	postID, err := strconv.Atoi(postIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Post.Unlike(c.Request.Context(), int64(postID), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) postsGetLiked(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.GetPostsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	posts, err := h.services.Post.FindUserLikes(c.Request.Context(), user.ID, input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, posts)
}
