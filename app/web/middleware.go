package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/muety/kitsquid/app/config"
	"strings"
)

const CLFDateFormat = "02/Jan/2006:15:04:05 -0700"

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

		tplCtx := getTplCtx(c)
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
		tplCtx := getTplCtx(c)
		c.Set(config.TemplateContextKey, &tplCtx)
		c.Next()
	}
}

func RemoteIp() gin.HandlerFunc {
	return func(c *gin.Context) {
		remoteIp := c.Request.RemoteAddr
		if ip := c.GetHeader("X-Real-Ip"); ip != "" {
			remoteIp = ip
		} else if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
			remoteIp = ip
		}

		if strings.Contains(remoteIp, ":") {
			remoteIp = strings.Split(remoteIp, ":")[0]
		}

		c.Set(config.RemoteIPKey, remoteIp)
		c.Next()
	}
}

func RequestCLFLogger(param gin.LogFormatterParams) string {
	// your custom format
	return fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d \n",
		param.ClientIP,
		param.TimeStamp.Format(CLFDateFormat),
		param.Method,
		param.Path,
		param.Request.Proto,
		param.StatusCode,
		param.BodySize,
	)
}
