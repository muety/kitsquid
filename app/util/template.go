package util

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kitsquid/app/common"
	"github.com/n1try/kitsquid/app/common/errors"
	"github.com/n1try/kitsquid/app/config"
)

func MakeError(c *gin.Context, tpl string, status int, error errors.KitSquidError, args *gin.H) {
	tplCtx := c.MustGet(config.TemplateContextKey)
	tplCtx.(*TplCtx).Errors = append(tplCtx.(*TplCtx).Errors, error.Error())

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

type TplCtx struct {
	User      interface{}
	Url       string
	Path      string
	Constants struct {
		FacultyIndex int
		VvzBaseUrl   string
	}
	Alerts       []string
	Errors       []string
	SemesterKeys []common.SemesterKey
}
