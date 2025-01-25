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

type postsGetReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (h *Handler) postsGetMy(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input postsGetReq
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

func (h *Handler) postsGetOne(c *gin.Context) {
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

	c.JSON(http.StatusOK, post)
}

func (h *Handler) postsGet(c *gin.Context) {
	var input postsGetReq
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
