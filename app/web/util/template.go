package util

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"github.com/n1try/kithub2/app/web/errors"
	"html/template"
	"strings"
)

func GetFuncMap() template.FuncMap {
	return template.FuncMap{
		"strIndex":    strIndex,
		"strRemove":   strRemove,
		"htmlSafe":    htmlSafe,
		"randomColor": util.RandomColor,
	}
}

type TplCtx struct {
	Path      string
	Constants struct {
		FacultyIndex int
		VvzBaseUrl   string
	}
	Alerts []string
	Errors []string
}

func GetTplCtx(c *gin.Context) TplCtx {
	var (
		alerts = make([]string, 0)
		errors = make([]string, 0)
	)

	if alert, ok := c.Request.URL.Query()["alert"]; ok {
		if msg, ok := config.Messages[alert[0]]; ok {
			alert = append(alert, msg)
		}
	}
	if err, ok := c.Request.URL.Query()["error"]; ok {
		if msg, ok := config.Messages[err[0]]; ok {
			errors = append(errors, msg)
		}
	}

	return TplCtx{
		Path: c.FullPath(),
		Constants: struct {
			FacultyIndex int
			VvzBaseUrl   string
		}{
			FacultyIndex: config.FacultyIdx,
			VvzBaseUrl:   config.KitVvzBaseUrl,
		},
		Alerts: alerts,
		Errors: errors,
	}
}

func MakeError(c *gin.Context, tpl string, status int, error errors.KitHubError, args *gin.H) {
	tplCtx := GetTplCtx(c)
	tplCtx.Errors = append(tplCtx.Errors, error.Error())

	h := gin.H{
		"tplCtx": tplCtx,
	}

	if args != nil {
		for k, v := range *args {
			h[k] = v
		}
	}

	c.HTML(status, tpl, h)
}

// Template funcs

func strIndex(x int, v string) string {
	return string([]rune(v)[:1])
}

func strRemove(html, needle string) string {
	return strings.ReplaceAll(html, needle, "")
}

func htmlSafe(html string) template.HTML {
	return template.HTML(html)
}
