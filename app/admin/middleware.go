package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kitsquid/app/common/errors"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/users"
	"net/http"
)

func checkAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, _ := c.Get(config.UserKey); user != nil && user.(*users.User).Admin {
			c.Next()
			return
		}
		c.AbortWithError(http.StatusUnauthorized, errors.Unauthorized{})
	}
}
