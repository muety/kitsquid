package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/n1try/kithub2/app/store"
	"github.com/n1try/kithub2/app/web/errors"
	"github.com/n1try/kithub2/app/web/util"
	"net/http"
)

func Index(c *gin.Context) {
	lectures, err := store.FindLectures(nil)
	if err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
		return
	}

	c.HTML(http.StatusOK, "index", gin.H{
		"lectures": lectures,
		"tplCtx":   util.GetTplCtx(c),
	})
}

func GetEvent(c *gin.Context) {
	lecture, err := store.GetLecture(c.Param("id"))
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePrivate)
		c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
		return
	}

	c.HTML(http.StatusOK, "event", gin.H{
		"lecture": lecture,
		"tplCtx":  util.GetTplCtx(c),
	})
}
