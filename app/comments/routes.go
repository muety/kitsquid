package comments

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/users"
	"github.com/n1try/kithub2/app/util"
	uuid "github.com/satori/go.uuid"
	"html/template"
	"net/http"
	"strings"
	"time"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.POST("/comments", CheckUser(), postComment(router))
	group.POST("/comments/delete", CheckUser(), deleteComment(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
}

func postComment(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var comment Comment

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		if err := c.ShouldBind(&comment); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		text := template.HTMLEscapeString(comment.Text)
		text = strings.ReplaceAll(text, "\r\n", "<br>")
		text = strings.ReplaceAll(text, "\n", "<br>")

		comment.Id = uuid.NewV4().String()
		comment.Text = text
		comment.Active = true // TODO: Admin functionality to activate comments
		comment.CreatedAt = time.Now()
		comment.UserId = user.Id
		if maxIdx, err := GetMaxIndexByEvent(comment.EventId); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		} else {
			comment.Index = maxIdx + 1
		}

		if err := Insert(&comment, false); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		// TODO: Prevent posting comments to non-existing events
		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    fmt.Sprintf("/event/%s", comment.EventId),
		})
	}
}

func deleteComment(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var comment commentDelete

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		if err := c.ShouldBind(&comment); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		existing, err := Get(comment.Id)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		if existing.UserId != user.Id {
			util.MakeError(c, "event", http.StatusUnauthorized, errors.Unauthorized{}, nil)
			return
		}

		if err := Delete(comment.Id); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    fmt.Sprintf("/event/%s", existing.EventId),
		})
	}
}

type commentDelete struct {
	Id string `form:"id" binding:"required"`
}
