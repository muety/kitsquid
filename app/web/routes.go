package web

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/events"
	"github.com/n1try/kithub2/app/users"
	"net/http"
)

func RegisterFallbackRoutes(router *gin.Engine) {
	router.NoMethod(ErrorHandle(), func(c *gin.Context) {
		c.AbortWithError(http.StatusMethodNotAllowed, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})

	router.NoRoute(ErrorHandle(), func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})
}

func RegisterStaticRoutes(router *gin.Engine) {
	router.Static("/assets", "app/public/build")
}

func RegisterMainRoutes(router *gin.Engine) *gin.RouterGroup {
	group := router.Group("/")
	group.Use(ErrorHandle())
	group.Use(users.ExtractUser())
	group.Use(TemplateContext())

	events.RegisterRoutes(router, group)
	users.RegisterRoutes(router, group)

	return group
}

func RegisterApiRoutes(router *gin.Engine) *gin.RouterGroup {
	group := router.Group("/api")
	group.Use(ApiErrorHandle())
	group.Use(users.ExtractUser())

	events.RegisterApiRoutes(router, group)
	users.RegisterApiRoutes(router, group)

	return group
}
