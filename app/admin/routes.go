package admin

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/muety/kitsquid/app/common/errors"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/events"
	"github.com/muety/kitsquid/app/scraping"
	"github.com/muety/kitsquid/app/users"
	"github.com/muety/kitsquid/app/util"
	"net/http"
	"strconv"
	"strings"
)

/*
RegisterRoutes registers all public routes with the given router instance
*/
func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/admin", checkAdmin(), getIndex(router))
}

/*
RegisterAPIRoutes registers all API routes with the given router instance
*/
func RegisterAPIRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.POST("/admin/query", checkAdmin(), apiAdminQuery(router))
	group.POST("/admin/flush", checkAdmin(), apiAdminFlush(router))
	group.POST("/admin/reindex", checkAdmin(), apiAdminReindex(router))
	group.POST("/admin/scrape", checkAdmin(), apiAdminScrape(router))
	group.POST("/admin/test_mail", checkAdmin(), apiAdminTestMail(router))
}

func getIndex(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {

		counters := map[string]int{
			"events":     events.Count(),
			"categories": events.CountCategories(),
			"faculties":  events.CountFaculties(),
			"bookmarks":  events.CountBookmarks(),
			"comments":   events.CountComments(),
			"reviews":    events.CountReviews(),
			"users":      users.Count(),
			"admins":     users.CountAdmins(),
		}

		c.HTML(http.StatusOK, "admin", gin.H{
			"entities": entities,
			"counters": counters,
			"tplCtx":   c.MustGet(config.TemplateContextKey),
		})
	}
}

func apiAdminQuery(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var re *registeredEntity
		var query adminQuery

		if err := c.ShouldBindJSON(&query); err != nil {
			c.Error(err)
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
			return
		}

		re, ok := registry[query.Entity]
		if !ok {
			c.AbortWithError(http.StatusNotFound, errors.NotFound{})
			return
		}

		var f func(*adminQuery, *registeredEntity, *gin.Context)

		switch strings.ToLower(query.Action) {
		case "list":
			f = handleList
			break
		case "get":
			f = handleGet
			break
		case "put":
			f = handlePut
			break
		case "delete":
			f = handleDelete
			break
		case "flush":
			f = handleFlush
			break
		case "reindex":
			f = handleReindex
		default:
			f = func(_q *adminQuery, _e *registeredEntity, context *gin.Context) {
				c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
				return
			}
		}

		f(&query, re, c)
	}
}

func apiAdminFlush(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		go func() {
			for _, e := range entities {
				if e.Resolvers.Flush != nil {
					go e.Resolvers.Flush()
				}
			}
		}()

		c.Status(http.StatusAccepted)
	}
}

func apiAdminReindex(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		go func() {
			for _, e := range entities {
				if e.Resolvers.Reindex != nil {
					go e.Resolvers.Reindex()
				}
			}
		}()

		c.Status(http.StatusAccepted)
	}
}

func apiAdminScrape(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var from int
		var to int
		var currentError error

		tguid := c.Request.URL.Query().Get("tguid")
		fromStr := c.Request.URL.Query().Get("from")
		toStr := c.Request.URL.Query().Get("to")

		if i, err := strconv.Atoi(fromStr); err == nil {
			from = i
		} else {
			currentError = err
		}
		if i, err := strconv.Atoi(toStr); err == nil {
			to = i
		} else {
			currentError = err
		}

		if tguid == "" || currentError != nil {
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
			return
		}

		go func(tguid string, from, to int) {
			log.Infoln("scraping start")

			mainScraper := scraping.NewEventScraper()
			mainJob := scraping.FetchEventsJob{
				Tguid: tguid,
				From:  from,
				To:    to,
			}
			mainResult, err := mainScraper.Run(mainJob)
			if err != nil {
				log.Errorf("event scraping failed – %v\n", err)
				return
			}

			if err := events.InsertMulti(mainResult.([]*events.Event), true, false); err != nil {
				log.Errorf("failed to store scraped events – %v\n", err)
				return
			}

			detailsScraper := scraping.NewEventDetailsScraper()
			detailsJob := scraping.FetchDetailsJob{Events: mainResult.([]*events.Event)}
			detailsResult, err := detailsScraper.Run(detailsJob)
			if err != nil {
				log.Errorf("event details scraping failed – %v\n", err)
				return
			}

			for _, l := range detailsResult.([]*events.Event) {
				if err := events.Insert(l, true, false); err != nil {
					log.Errorf("failed to update event %s – %v\n", l.Id, err)
				}
			}

			log.Infoln("scraping end")
		}(tguid, from, to)

		c.Status(http.StatusAccepted)
	}
}

func apiAdminTestMail(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		to := c.Request.URL.Query().Get("to")
		if to == "" {
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
			return
		}

		if err := util.SendTestMail(to); err != nil {
			c.Error(err)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
			return
		}

		c.Status(http.StatusOK)
	}
}

func handleList(q *adminQuery, re *registeredEntity, c *gin.Context) {
	result, err := re.Resolvers.List()
	if err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleGet(q *adminQuery, re *registeredEntity, c *gin.Context) {
	if q.Key == "" {
		c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
		return
	}

	result, err := re.Resolvers.Get(q.Key)
	if err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handlePut(q *adminQuery, re *registeredEntity, c *gin.Context) {
	if q.Key == "" || q.Value == "" {
		c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
		return
	}

	if err := re.Resolvers.Put(q.Key, q.Value); err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
		return
	}

	c.Status(http.StatusOK)
}

func handleDelete(q *adminQuery, re *registeredEntity, c *gin.Context) {
	if q.Key == "" {
		c.AbortWithError(http.StatusBadRequest, errors.BadRequest{})
		return
	}

	if err := re.Resolvers.Delete(q.Key); err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleFlush(q *adminQuery, re *registeredEntity, c *gin.Context) {
	if re.Resolvers.Flush != nil {
		go re.Resolvers.Flush()
	}
	c.Status(http.StatusAccepted)
}

func handleReindex(q *adminQuery, re *registeredEntity, c *gin.Context) {
	if re.Resolvers.Reindex != nil {
		go re.Resolvers.Reindex()
	}
	c.Status(http.StatusAccepted)
}
