package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) notRequiredAuthMiddleware(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.Next()
		return
	}

	accessToken := strings.Split(header, " ")[1]
	if accessToken == "" {
		c.Next()
		return
	}

	user, err := h.getUserDataFromAccessTokenClaims(c.Request.Context(), accessToken)
	if err != nil {
		c.Next()
		return
	}

	c.Set("cached-user", *user)

	c.Next()
}
