package web

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/config"
)

func ErrorHandle() gin.HandlerFunc {
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

		tplCtx := GetTplCtx(c)
		tplCtx.Errors = errors

		c.HTML(c.Writer.Status(), "empty", gin.H{
			"tplCtx": tplCtx,
		})

		return
	}
}

func ApiErrorHandle() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || !c.IsAborted() {
			return
		}

		if c.Errors.ByType(gin.ErrorTypePublic).Last() != nil {
			c.JSON(c.Writer.Status(), map[string]string{
				"error": c.Errors.ByType(gin.ErrorTypePublic).Last().Error(),
			})
		} else {
			c.Status(c.Writer.Status())
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
