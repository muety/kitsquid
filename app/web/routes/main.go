package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/model"
	"github.com/n1try/kithub2/app/store"
	"github.com/n1try/kithub2/app/util"
	"github.com/n1try/kithub2/app/web/errors"
	webutil "github.com/n1try/kithub2/app/web/util"
	"github.com/timshannon/bolthold"
	"net/http"
)

func Index(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		events, err := store.FindEvents(nil)
		if err != nil {
			c.Error(err)
			webutil.MakeError(c, "index", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"events": events,
			"tplCtx": webutil.GetTplCtx(c),
		})
	}
}

func GetEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		event, err := store.GetEvent(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			webutil.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"event":  event,
			"tplCtx": webutil.GetTplCtx(c),
		})
	}
}

func GetSignup(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		cfg := config.Get()

		c.HTML(http.StatusOK, "signup", gin.H{
			"whitelist":  cfg.Auth.Whitelist,
			"university": cfg.University,
			"tplCtx":     webutil.GetTplCtx(c),
		})
	}
}

func PostSignup(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		cfg := config.Get()

		var user model.User

		h := &gin.H{
			"whitelist":  cfg.Auth.Whitelist,
			"university": cfg.University,
		}

		if err := c.ShouldBind(&user); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			webutil.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if !user.IsValid(util.ValidateUser) {
			webutil.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if err := util.HashPassword(&user); err != nil {
			webutil.MakeError(c, "signup", http.StatusInternalServerError, errors.Internal{}, h)
			return
		}

		if err := store.InsertUser(&user, false); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			if err == bolthold.ErrKeyExists {
				webutil.MakeError(c, "signup", http.StatusConflict, errors.Conflict{}, h)
			} else {
				webutil.MakeError(c, "signup", http.StatusInternalServerError, errors.Internal{}, h)
			}
			return
		}

		c.Request.URL.Path = "/"
		c.Request.URL.RawQuery = "alert=signup_success"
		c.Request.Method = http.MethodGet
		r.HandleContext(c)
	}
}
