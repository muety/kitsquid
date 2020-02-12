package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/web/util"
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

		if len(c.Errors) == 0 {
			return
		}

		var errors = make([]string, 0)
		for _, e := range c.Errors {
			if e.Type == gin.ErrorTypePublic || cfg.IsDev() {
				errors = append(errors, e.Error())
			}
		}

		tplCtx := util.GetTplCtx(c)
		tplCtx.Errors = errors

		c.HTML(c.Writer.Status(), "error", gin.H{
			"tplCtx": tplCtx,
		})
		return
	}
}
