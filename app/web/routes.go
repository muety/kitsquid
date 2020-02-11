package web

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/web/errors"
	"github.com/n1try/kithub2/app/web/routes"
	"net/http"
)

func RegisterFallbackRoutes(router *gin.Engine) {
	router.NoMethod(func(c *gin.Context) {
		c.AbortWithError(http.StatusMethodNotAllowed, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})

	router.NoRoute(func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})
}

func RegisterStaticRoutes(router *gin.Engine) {
	router.Static("/assets", "app/public/build")
}

func RegisterMainRoutes(router *gin.Engine) *gin.RouterGroup {
	group := router.Group("/")
	group.Use(AssetsPush())

	group.GET("/", routes.Index)
	group.GET("/event/:id", routes.GetEvent)

	return group
}
