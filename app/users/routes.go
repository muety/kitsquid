package users

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	uuid "github.com/satori/go.uuid"
	"github.com/timshannon/bolthold"
	"net/http"
	"time"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/signup", getSignup(router))
	group.POST("/signup", postSignup(router))
	group.GET("/login", getLogin(router))
	group.POST("/login", postLogin(router))
	group.POST("/logout", postLogout(router))
	group.GET("/activate", apiGetActivate(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
}

func postLogout(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		sess, ok := c.Get(config.SessionKey)
		if !ok {
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		DeleteSession(sess.(*UserSession))

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=logout_success",
		})
	}
}

func getLogin(r *gin.Engine) func(c *gin.Context) {
	cfg := config.Get()

	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "login", gin.H{
			"whitelist": cfg.Auth.Whitelist,
			"tplCtx":    c.MustGet(config.TemplateContextKey),
		})
	}
}

func postLogin(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var l Login

		h := &gin.H{
			"whitelist": cfg.Auth.Whitelist,
		}

		if err := c.ShouldBind(&l); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "login", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		user, err := Get(l.UserId)
		if err != nil || !CheckPasswordHash(user, l.Password) || !user.Active {
			if err != nil {
				c.Error(err).SetType(gin.ErrorTypePrivate)
			}
			util.MakeError(c, "login", http.StatusUnauthorized, errors.Unauthorized{}, h)
			return
		}

		sess := &UserSession{
			Token:     uuid.NewV4().String(),
			UserId:    user.Id,
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		if err := InsertSession(sess, false); err != nil {
			util.MakeError(c, "login", http.StatusInternalServerError, errors.Internal{}, h)
			return
		}

		c.SetCookie(config.SessionKey,
			sess.Token,
			int(cfg.SessionTimeout().Seconds()),
			"/",
			"",
			!cfg.IsDev(),
			true)

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/",
		})
	}
}

func getSignup(r *gin.Engine) func(c *gin.Context) {
	cfg := config.Get()

	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "signup", gin.H{
			"whitelist":  cfg.Auth.Whitelist,
			"university": cfg.University,
			"tplCtx":     c.MustGet(config.TemplateContextKey),
		})
	}
}

func postSignup(r *gin.Engine) func(c *gin.Context) {
	cfg := config.Get()
	validator := NewUserValidator(config.Get())

	return func(c *gin.Context) {
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

		if !user.IsValid(validator) {
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

		// TODO: Rollback user creation if token creation fails
		activationToken := uuid.NewV4().String()
		if err := InsertToken(activationToken, user.Id); err != nil {
			util.MakeError(c, "signup", http.StatusInternalServerError, errors.Internal{}, h)
			return
		}

		go func(user *User, token string) {
			if err := SendConfirmationMail(user, token); err != nil {
				log.Errorf("failed to send confirmation mail to %s – %v", user.Id, err)
			}
		}(&user, activationToken)

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=signup_success",
		})
	}
}

func apiGetActivate(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		makeError := func() {
			c.HTML(http.StatusNotFound, "redirect", gin.H{
				"tplCtx": c.MustGet(config.TemplateContextKey),
				"url":    "/?error=activate_failure",
			})
		}

		token := c.Request.URL.Query().Get("token")
		if token == "" {
			makeError()
			return
		}

		userId, err := GetToken(token)
		if err != nil {
			c.Error(err)
			makeError()
			return
		}

		user, err := Get(userId)
		if err != nil {
			c.Error(err)
			makeError()
			return
		}

		user.Active = true
		if err := Insert(user, true); err != nil {
			c.Error(err)
			makeError()
			return
		}

		go func(token, userId string) {
			if err := DeleteToken(token); err != nil {
				log.Errorf("failed to delete token for %s", userId)
				return
			}
		}(token, userId)

		c.HTML(http.StatusNotFound, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=activate_success",
		})
	}
}
