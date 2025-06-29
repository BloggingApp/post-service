package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/BloggingApp/post-service/internal/dto"
	"github.com/gin-gonic/gin"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
)

func (h *Handler) moderatorMiddleware(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, errNotAuthorized.Error()))
		c.Abort()
		return
	}

	accessToken := strings.Split(header, " ")[1]
	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, errNotAuthorized.Error()))
		c.Abort()
		return
	}

	claims, err := jwtmanager.DecodeJWT(accessToken, []byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, errNotAuthorized.Error()))
		c.Abort()
		return
	}

	role := strings.ToLower(claims["role"].(string))
	if role != "mod" && role != "admin" {
		c.JSON(http.StatusForbidden, dto.NewBasicResponse(false, "no access"))
		c.Abort()
		return
	}

	user, err := h.getUserDataFromClaims(c.Request.Context(), claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		c.Abort()
		return
	}

	c.Set("cached-user", *user)

	c.Next()
}
