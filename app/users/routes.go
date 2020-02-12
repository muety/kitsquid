package users

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"github.com/timshannon/bolthold"
	"net/http"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/signup", getSignup(router))
	group.POST("/signup", postSignup(router))
}

func getSignup(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		cfg := config.Get()

		c.HTML(http.StatusOK, "signup", gin.H{
			"whitelist":  cfg.Auth.Whitelist,
			"university": cfg.University,
			"tplCtx":     util.GetTplCtx(c),
		})
	}
}

func postSignup(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		cfg := config.Get()

		var user User

		h := &gin.H{
			"whitelist":  cfg.Auth.Whitelist,
			"university": cfg.University,
		}

		if err := c.ShouldBind(&user); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if !user.IsValid(Validate) {
			util.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if err := HashPassword(&user); err != nil {
			util.MakeError(c, "signup", http.StatusInternalServerError, errors.Internal{}, h)
			return
		}

		if err := Insert(&user, false); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			if err == bolthold.ErrKeyExists {
				util.MakeError(c, "signup", http.StatusConflict, errors.Conflict{}, h)
			} else {
				util.MakeError(c, "signup", http.StatusInternalServerError, errors.Internal{}, h)
			}
			return
		}

		c.HTML(http.StatusOK, "post_signup", gin.H{
			"tplCtx": util.GetTplCtx(c),
		})
	}
}
