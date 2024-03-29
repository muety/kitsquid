package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/events"
	"github.com/muety/kitsquid/app/users"
	"github.com/muety/kitsquid/app/util"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func getFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":           add,
		"in":            in,
		"strIndex":      strIndex,
		"strRemove":     strRemove,
		"strSplit":      strings.Split,
		"strPrefix":     strings.HasPrefix,
		"strCapitalize": strings.Title,
		"htmlSafe":      htmlSafe,
		"randomColor":   util.RandomColor,
		"paginate":      paginate,
		"date":          formatDate,
		"date3339":      formatDate3339,
		"noescape":      noescape,
	}
}

func getTplCtx(c *gin.Context) util.TplCtx {
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

	semesters, _ := events.GetSemesters()

	return util.TplCtx{
		Url:  c.Request.URL.String(),
		Path: c.FullPath(),
		User: user,
		Constants: struct {
			FacultyIndex int
			VvzBaseUrl   string
		}{
			FacultyIndex: config.FacultyIdx,
			VvzBaseUrl:   config.KitVvzBaseURL,
		},
		Alerts:       alerts,
		Errors:       errors,
		SemesterKeys: semesters,
		Version:      config.Get().Version,
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

func paginate(path string, direction int) string {
	u, err := url.Parse(path)
	if err != nil {
		return ""
	}

	var (
		limit  = config.Get().Misc.Pagesize
		offset = 0
	)

	if l, err := strconv.Atoi(u.Query().Get("limit")); err == nil {
		limit = l
	}
	if o, err := strconv.Atoi(u.Query().Get("offset")); err == nil {
		offset = o
	}

	if direction < 0 {
		offset = int(math.Max(0, float64(offset-limit)))
	} else {
		offset += limit
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	return u.String()
}

func formatDate(t time.Time) string {
	return t.Format(time.RFC822)
}

func formatDate3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

func noescape(s string) template.HTML {
	return template.HTML(s)
}

func in(needle interface{}, haystack ...interface{}) bool {
	for _, e := range haystack {
		if e == needle {
			return true
		}
	}
	return false
}

func health() uint8 {
	var dbUp uint8
	if _, err := users.Get(cfg.Auth.Admin.User); err == nil {
		dbUp = 1
	}
	return dbUp
}
