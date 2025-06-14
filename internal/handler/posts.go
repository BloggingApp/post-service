package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) postsUploadImage(c *gin.Context) {
	file, fileHeader, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	url, err := h.services.Post.UploadTempPostImage(c.Request.Context(), file, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, url)
}

func (h *Handler) postsCreate(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.CreatePostRequest
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	createdPost, err := h.services.Post.Create(c.Request.Context(), user.ID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, *createdPost)
}

func (h *Handler) postsGetMy(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.GetPostsRequest
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
	var input dto.GetPostsRequest
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

	if err := h.services.Post.Like(c.Request.Context(), int64(postID), user.ID, false); err != nil {
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

	if err := h.services.Post.Like(c.Request.Context(), int64(postID), user.ID, true); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) postsGetLiked(c *gin.Context) {
	user := h.getCachedUserFromRequest(c)

	var input dto.GetPostsRequest
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

func (h *Handler) postsTrending(c *gin.Context) {
	hours, err0 := strconv.Atoi(c.Query("hours"))
	limit, err1 := strconv.Atoi(c.Query("limit"))
	if err0 != nil || err1 != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errHoursAndLimitMustBeInt.Error()))
		return
	}

	posts, err := h.services.Post.GetTrending(c.Request.Context(), hours, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *Handler) postsSearchByTitle(c *gin.Context) {
	limit, err0 := strconv.Atoi(c.Query("limit"))
	offset, err1 := strconv.Atoi(c.Query("offset"))
	if err0 != nil || err1 != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errLimitAndOffsetMustBeInt.Error()))
		return
	}
	title := c.Query("q")

	result, err := h.services.Post.SearchByTitle(c.Request.Context(), title, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}
