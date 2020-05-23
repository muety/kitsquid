package web

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kitsquid/app/admin"
	"github.com/n1try/kitsquid/app/common/errors"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/events"
	"github.com/n1try/kitsquid/app/users"
	"net/http"
)

func registerFallbackRoutes(router *gin.Engine) {
	router.NoMethod(ErrorHandle(), func(c *gin.Context) {
		c.AbortWithError(http.StatusMethodNotAllowed, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})

	router.NoRoute(ErrorHandle(), func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})
}

func registerStaticRoutes(router *gin.Engine) {
	router.Static("/assets", "app/public/build")
	router.StaticFile("favicon.ico", "app/public/build/favicon.ico")
}

func registerMainRoutes(router *gin.Engine) *gin.RouterGroup {
	group := router.Group("/")
	group.Use(ErrorHandle())
	group.Use(users.ExtractUser())
	group.Use(TemplateContext())

	events.RegisterRoutes(router, group)
	users.RegisterRoutes(router, group)
	admin.RegisterRoutes(router, group)

	group.GET("/about", func(c *gin.Context) {
		c.HTML(http.StatusOK, "about", gin.H{"tplCtx": c.MustGet(config.TemplateContextKey)})
	})
	group.GET("/imprint", func(c *gin.Context) {
		c.HTML(http.StatusOK, "imprint", gin.H{"tplCtx": c.MustGet(config.TemplateContextKey)})
	})
	group.GET("/data-privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "data_privacy", gin.H{"tplCtx": c.MustGet(config.TemplateContextKey)})
	})
	group.GET("/health", func(c *gin.Context) {
		c.String(200, "app=1\ndb=%d", health())
	})

	return group
}

func registerAPIRoutes(router *gin.Engine) *gin.RouterGroup {
	group := router.Group("/api")
	group.Use(ApiErrorHandle())
	group.Use(users.ExtractUser())

	events.RegisterAPIRoutes(router, group)
	users.RegisterAPIRoutes(router, group)
	admin.RegisterAPIRoutes(router, group)

	return group
}
