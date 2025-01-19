package handler

import (
	"net/http"
	"strconv"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/gin-gonic/gin"
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
