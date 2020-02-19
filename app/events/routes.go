package events

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/users"
	"github.com/n1try/kithub2/app/util"
	"net/http"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/", getEvents(router))
	group.GET("/event/:id", getEvent(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.PUT("/event/:id/bookmark", apiPutBookmark(router))
}

func getEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		events, err := FindAll(nil)
		if err != nil {
			c.Error(err)
			util.MakeError(c, "index", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"events": events,
			"tplCtx": c.MustGet(config.TemplateContextKey),
		})
	}
}

func getEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		semester := common.SemesterKeys[len(common.SemesterKeys)-1]
		if s := c.Request.URL.Query().Get("semester"); s != "" {
			semester = common.SemesterKey(s)
		}

		var bookmarked bool
		var user *users.User
		if u, ok := c.Get(config.UserKey); ok {
			user = u.(*users.User)
			if _, err := FindBookmark(user.Id, event.Id); err == nil {
				bookmarked = true
			}
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"event":         event,
			"bookmarked":    bookmarked,
			"semesterQuery": semester,
			"tplCtx":        c.MustGet(config.TemplateContextKey),
		})
	}
}

func apiPutBookmark(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var user *users.User
		if u, ok := c.Get(config.UserKey); !ok {
			c.AbortWithError(http.StatusUnauthorized, errors.Unauthorized{}).SetType(gin.ErrorTypePublic)
			return
		} else {
			user = u.(*users.User)
		}

		event, err := Get(c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
			return
		}

		bm, err := FindBookmark(user.Id, event.Id)
		if err != nil {
			if err := InsertBookmark(&Bookmark{
				UserId:   user.Id,
				EntityId: event.Id,
			}); err != nil {
				c.Error(err)
				c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
				return
			}

			c.Status(http.StatusCreated)
		} else {
			if err := DeleteBookmark(bm); err != nil {
				c.Error(err)
				c.AbortWithError(500, errors.Internal{}).SetType(gin.ErrorTypePublic)
				return
			}

			c.Status(http.StatusNoContent)
		}
	}
}
