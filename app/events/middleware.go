package events

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"net/http"
)

func CheckUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Next()

		user, _ := c.Get(config.UserKey)
		if user == nil {
			c.AbortWithError(http.StatusUnauthorized, errors.Unauthorized{})
			return
		}
	}
}
