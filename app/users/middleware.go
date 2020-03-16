package users

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"net/http"
	"time"
)

func ExtractUser() gin.HandlerFunc {
	validator := NewSessionValidator(config.Get(), Get)

	return func(c *gin.Context) {
		defer c.Next()

		token, err := c.Cookie(config.SessionKey)
		if err != nil {
			return
		}
		sess, err := GetSession(token)
		if err != nil {
			return
		}

		if !sess.IsValid(validator) {
			return
		}

		user, _ := Get(sess.UserId)
		c.Set(config.UserKey, user)
		c.Set(config.SessionKey, sess)

		if c.Request.URL.Path == "/logout" {
			return
		}

		sessNew := *sess
		sessNew.LastSeen = time.Now()

		go func() {
			if err := InsertSession(&sessNew, true); err != nil {
				log.Errorf("failed to update session – %v\n", err)
			}
		}()
	}
}

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
