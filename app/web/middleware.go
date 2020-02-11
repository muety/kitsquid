package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/web/errors"
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
	return func(c *gin.Context) {
		c.Next()

		var error errors.KitHubError
		e := c.Errors.ByType(gin.ErrorTypePublic).Last()
		if e == nil {
			if e := c.Errors.Last(); e == nil {
				return
			}
			error = errors.Internal{}
		} else {
			error = e.Err
		}

		c.HTML(c.Writer.Status(), "error", gin.H{
			"error":  error.Error(),
			"tplCtx": util.GetTplCtx(c),
		})
		return
	}
}
