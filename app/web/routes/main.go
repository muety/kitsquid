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
		lectures, err := store.FindLectures(nil)
		if err != nil {
			c.Error(err)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"lectures": lectures,
			"tplCtx":   webutil.GetTplCtx(c),
		})
	}
}

func GetEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		lecture, err := store.GetLecture(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
			return
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"lecture": lecture,
			"tplCtx":  webutil.GetTplCtx(c),
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
		var user model.User

		if err := c.ShouldBind(&user); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{}).SetType(gin.ErrorTypePublic)
			return
		}

		if !user.IsValid(util.ValidateUser) {
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{}).SetType(gin.ErrorTypePublic)
			return
		}

		if err := util.HashPassword(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		if err := store.InsertUser(&user, false); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			if err == bolthold.ErrKeyExists {
				c.AbortWithError(http.StatusConflict, errors.Conflict{}).SetType(gin.ErrorTypePublic)
			} else {
				c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			}
			return
		}

		c.Request.URL.Path = "/"
		c.Request.URL.RawQuery = "postsignup"
		c.Request.Method = http.MethodGet
		r.HandleContext(c)
	}
}
