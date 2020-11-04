package events

import (
	"github.com/gin-gonic/gin"
	"github.com/muety/kitsquid/app/common/errors"
	"github.com/muety/kitsquid/app/config"
	"net/http"
)

/*
CheckUser checks whether a user is attached to the current context
*/
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
