package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) commentsCreate(c *gin.Context) {
	user := h.getUserFromRequest(c)

	var input dto.CreateCommentDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	createdComment, err := h.services.Comment.Create(c.Request.Context(), user.ID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, createdComment)
}

func (h *Handler) commentsGet(c *gin.Context) {
	postIDString := strings.TrimSpace(c.Param("postID"))
	postID, err := strconv.Atoi(postIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidPostID.Error()))
		return
	}

	var input dto.GetCommentsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	comments, err := h.services.Comment.FindPostComments(c.Request.Context(), int64(postID), input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, comments)
}

func (h *Handler) commentsGetReplies(c *gin.Context) {
	postIDString := strings.TrimSpace(c.Param("postID"))
	postID, err0 := strconv.Atoi(postIDString)

	commentIDString := strings.TrimSpace(c.Param("commentID"))
	commentID, err1 := strconv.Atoi(commentIDString)
	
	if err0 != nil || err1 != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidID.Error()))
		return
	}

	var input dto.GetCommentsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	replies, err := h.services.Comment.FindCommentReplies(c.Request.Context(), int64(postID), int64(commentID), input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, replies)
}

func (h *Handler) commentsDelete(c *gin.Context) {
	user := h.getUserFromRequest(c)

	postIDString := strings.TrimSpace(c.Param("postID"))
	postID, err0 := strconv.Atoi(postIDString)

	commentIDString := strings.TrimSpace(c.Param("commentID"))
	commentID, err1 := strconv.Atoi(commentIDString)
	
	if err0 != nil || err1 != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidID.Error()))
		return
	}

	if err := h.services.Comment.Delete(c.Request.Context(), int64(postID), int64(commentID), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) commentsIsLiked(c *gin.Context) {
	user := h.getUserFromRequest(c)

	commentIDString := strings.TrimSpace(c.Param("commentID"))
	commentID, err := strconv.Atoi(commentIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	isLiked := h.services.Comment.IsLiked(c.Request.Context(), int64(commentID), user.ID)

	c.JSON(http.StatusOK, gin.H{"isLiked": isLiked})
}

func (h *Handler) commentsLike(c *gin.Context) {
	user := h.getUserFromRequest(c)

	commentIDString := strings.TrimSpace(c.Param("commentID"))
	commentID, err := strconv.Atoi(commentIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Comment.Like(c.Request.Context(), int64(commentID), user.ID, false); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) commentsUnlike(c *gin.Context) {
	user := h.getUserFromRequest(c)

	commentIDString := strings.TrimSpace(c.Param("commentID"))
	commentID, err := strconv.Atoi(commentIDString)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Comment.Like(c.Request.Context(), int64(commentID), user.ID, true); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}
