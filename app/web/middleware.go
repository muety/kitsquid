package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"strings"
)

func AssetsPush() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if pusher := c.Writer.Pusher(); pusher != nil {
			for _, a := range config.PushAssets {
				if err := pusher.Push(a, nil); err != nil {
					glog.Errorf("failed to push %s – %v", a, err)
				}
			}
		}
	}
}

func ErrorHandler() gin.HandlerFunc {
	cfg := config.Get()

	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || !c.IsAborted() {
			return
		}

		var errors = make([]string, 0)
		for _, e := range c.Errors {
			if e.Type == gin.ErrorTypePublic || cfg.IsDev() {
				errors = append(errors, e.Error())
			}
		}

		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			err := c.Errors.ByType(gin.ErrorTypePublic).Last()
			c.JSON(c.Writer.Status(), map[string]string{
				"error": err.Error(),
			})
		} else {
			tplCtx := GetTplCtx(c)
			tplCtx.Errors = errors

			c.HTML(c.Writer.Status(), "empty", gin.H{
				"tplCtx": tplCtx,
			})
		}

		return
	}
}

func TemplateContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tplCtx := GetTplCtx(c)
		c.Set(config.TemplateContextKey, &tplCtx)
		c.Next()
	}
}
