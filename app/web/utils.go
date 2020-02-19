package web

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/common"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/users"
	"github.com/n1try/kithub2/app/util"
	"html/template"
	"strings"
)

func GetFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":         add,
		"strIndex":    strIndex,
		"strRemove":   strRemove,
		"strSplit":    strings.Split,
		"strPrefix":   strings.HasPrefix,
		"htmlSafe":    htmlSafe,
		"randomColor": util.RandomColor,
	}
}

func GetTplCtx(c *gin.Context) util.TplCtx {
	var (
		alerts = make([]string, 0)
		errors = make([]string, 0)
	)

	if alert, ok := c.Request.URL.Query()["alert"]; ok {
		if msg, ok := config.Messages[alert[0]]; ok {
			alerts = append(alerts, msg)
		}
	}
	if err, ok := c.Request.URL.Query()["error"]; ok {
		if msg, ok := config.Messages[err[0]]; ok {
			errors = append(errors, msg)
		}
	}

	var user *users.User
	if u, ok := c.Get(config.UserKey); ok {
		user = u.(*users.User)
	}

	return util.TplCtx{
		Path: c.FullPath(),
		User: user,
		Constants: struct {
			FacultyIndex int
			VvzBaseUrl   string
		}{
			FacultyIndex: config.FacultyIdx,
			VvzBaseUrl:   config.KitVvzBaseUrl,
		},
		Alerts:       alerts,
		Errors:       errors,
		SemesterKeys: common.SemesterKeys,
	}
}

func strIndex(x int, v string) string {
	return string([]rune(v)[:1])
}

func strRemove(str, needle string) string {
	return strings.ReplaceAll(str, needle, "")
}

func add(a, b int) int {
	return a + b
}

func htmlSafe(html string) template.HTML {
	return template.HTML(html)
}
