package events

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"net/http"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/", getEvents(router))
	group.GET("/event/:id", getEvent(router))
}

func getEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		tplCtx, _ := c.Get(config.TemplateContextKey)

		events, err := FindAll(nil)
		if err != nil {
			c.Error(err)
			util.MakeError(c, "index", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"events": events,
			"tplCtx": tplCtx,
		})
	}
}

func getEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		tplCtx, _ := c.Get(config.TemplateContextKey)

		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"event":  event,
			"tplCtx": tplCtx,
		})
	}
}
