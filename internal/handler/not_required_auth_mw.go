package handler

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
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

	claims, err := jwtmanager.DecodeJWT(accessToken, []byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		c.Next()
		return
	}

	user, err := h.getUserDataFromClaims(c.Request.Context(), claims)
	if err != nil {
		c.Next()
		return
	}

	c.Set("user", *user)

	c.Next()
}
