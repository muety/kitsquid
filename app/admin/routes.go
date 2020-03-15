package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/config"
	"net/http"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/admin", CheckAdmin(), index(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
}

func index(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
		})
	}
}
