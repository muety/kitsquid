package users

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/leandro-lugaresi/hub"
	"github.com/muety/kitsquid/app/common/errors"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/util"
	uuid "github.com/satori/go.uuid"
	"github.com/timshannon/bolthold"
	"net/http"
	"time"
)

/*
RegisterRoutes registers all public routes with the given router instance
*/
func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/signup", getSignup(router))
	group.GET("/login", getLogin(router))
	group.GET("/account", CheckUser(), getAccount(router))
	group.GET("/activate", getActivate(router))

	group.POST("/signup", postSignup(router))
	group.POST("/login", postLogin(router))
	group.POST("/account", CheckUser(), postAccount(router))
	group.POST("/account/delete", CheckUser(), postDeleteAccount(router))
	group.POST("/logout", CheckUser(), postLogout(router))
}

/*
RegisterAPIRoutes registers all API routes with the given router instance
*/
func RegisterAPIRoutes(router *gin.Engine, group *gin.RouterGroup) {
}

func postLogout(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		sess, ok := c.Get(config.SessionKey)
		if !ok {
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		_ = DeleteSession(sess.(*UserSession))

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
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "login", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		user, err := Get(l.UserId)
		if err != nil || !CheckPasswordHash(user, l.Password) || !user.Active {
			if err != nil {
				_ = c.Error(err).SetType(gin.ErrorTypePrivate)
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
			"whitelist":    cfg.Auth.Whitelist,
			"university":   cfg.University,
			"grecaptchaId": cfg.Recaptcha.ClientID,
			"tplCtx":       c.MustGet(config.TemplateContextKey),
		})
	}
}

func postSignup(r *gin.Engine) func(c *gin.Context) {
	cfg := config.Get()
	validator := NewUserValidator(cfg, true)

	return func(c *gin.Context) {
		var user User
		var recaptcha recaptchaClientRequest

		h := &gin.H{
			"whitelist":    cfg.Auth.Whitelist,
			"university":   cfg.University,
			"grecaptchaId": cfg.Recaptcha.ClientID,
		}

		if err := c.ShouldBind(&recaptcha); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if !ValidateRecaptcha(recaptcha.GRecaptchaToken, c.GetString(config.RemoteIPKey)) {
			log.Errorf("recaptcha validation failed while trying to sign up user %s\n", user.Id)
			util.MakeError(c, "signup", http.StatusBadRequest, errors.BadRequest{}, h)
			return
		}

		if err := c.ShouldBind(&user); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
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

		user.CreatedAt = time.Now()
		if u, err := Get(user.Id); err != nil || !u.Admin {
			user.Admin = false
		}

		if err := Insert(&user, false); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
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
				log.Errorf("failed to send confirmation mail to %s – %v\n", user.Id, err)
			}
		}(&user, activationToken)

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=signup_success",
		})
	}
}

func getAccount(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "account", gin.H{
			"university": cfg.University,
			"tplCtx":     c.MustGet(config.TemplateContextKey),
		})
	}
}

func postAccount(r *gin.Engine) func(c *gin.Context) {
	cfg := config.Get()
	userValidator := NewUserValidator(cfg, false)
	pwValidator := NewUserCredentialsValidator(cfg, true)

	return func(c *gin.Context) {
		var change accountChange

		u, _ := c.Get(config.UserKey)
		user := u.(*User)
		user.Admin = false

		if err := c.ShouldBind(&change); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "account", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		if change.OldPassword != "" && change.NewPassword != "" {
			if !CheckPasswordHash(user, change.OldPassword) {
				util.MakeError(c, "account", http.StatusUnauthorized, errors.Unauthorized{}, nil)
				return
			}

			if !user.HasValidCredentials(pwValidator) {
				util.MakeError(c, "account", http.StatusBadRequest, errors.BadRequest{}, nil)
				return
			}

			user.Password = change.NewPassword
			if err := HashPassword(user); err != nil {
				util.MakeError(c, "account", http.StatusInternalServerError, errors.Internal{}, nil)
				return
			}
		}

		if change.Gender != "" {
			user.Gender = change.Gender
		}
		if change.Major != "" {
			user.Major = change.Major
		}
		if change.Degree != "" {
			user.Degree = change.Degree
		}

		if !user.IsValid(userValidator) {
			util.MakeError(c, "account", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		if err := Insert(user, true); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "account", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/account?alert=account_change_success",
		})
	}
}

func postDeleteAccount(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		u, _ := c.Get(config.UserKey)
		user := u.(*User)

		s, _ := c.Get(config.SessionKey)
		session := s.(*UserSession)

		if err := Delete(user.Id); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "account", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		if err := DeleteSession(session); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypePrivate)
		}

		config.EventBus().Publish(hub.Message{
			Name:   config.EventAccountDelete,
			Fields: hub.Fields{"id": user.Id},
		})

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=account_delete_success",
		})
	}
}

func getActivate(r *gin.Engine) func(c *gin.Context) {
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

		userID, err := GetToken(token)
		if err != nil {
			_ = c.Error(err)
			makeError()
			return
		}

		user, err := Get(userID)
		if err != nil {
			_ = c.Error(err)
			makeError()
			return
		}

		user.Active = true
		user.Admin = false
		if err := Insert(user, true); err != nil {
			_ = c.Error(err)
			makeError()
			return
		}

		go func(token, userId string) {
			if err := DeleteToken(token); err != nil {
				log.Errorf("failed to delete token for %s\n", userId)
				return
			}
		}(token, userID)

		c.HTML(http.StatusNotFound, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    "/?alert=activate_success",
		})
	}
}
