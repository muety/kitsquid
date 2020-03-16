package events

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/comments"
	"github.com/n1try/kithub2/app/common"
	"github.com/n1try/kithub2/app/common/errors"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/reviews"
	"github.com/n1try/kithub2/app/users"
	"github.com/n1try/kithub2/app/util"
	"net/http"
	"net/url"
	"strconv"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/", getEvents(router))
	group.GET("/bookmarks", CheckUser(), getBookmarks(router))
	group.GET("/event/:id", getEvent(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/event/search", apiSearchEvents(router))
	group.PUT("/event/:id/bookmark", CheckUser(), apiPutBookmark(router))
}

func getEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		eventQuery := buildEventQuery(c.Request.URL.Query())
		events, err := Find(eventQuery)
		if err != nil {
			c.Error(err)
			util.MakeError(c, "index", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		categories, _ := GetCategories()
		types, _ := GetTypes()
		lecturers, _ := GetLecturers()

		eventRatings := make(map[string]float32)
		for _, e := range events {
			eventRatings[e.Id] = 0
			if averages, err := reviews.GetAverages(e.Id); err == nil {
				if avg, ok := averages[reviews.KeyMainRating]; ok {
					eventRatings[e.Id] = avg
				}
			}
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"events":       events,
			"types":        types,
			"eventRatings": eventRatings,
			"categories":   categories,
			"lecturers":    lecturers,
			"limit":        eventQuery.Limit,
			"offset":       eventQuery.Skip,
			"tplCtx":       c.MustGet(config.TemplateContextKey),
		})
	}
}

func getBookmarks(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		bookmarks, err := FindBookmarks(user.Id)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "bookmarks", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		events := make([]*Event, len(bookmarks))
		for i, b := range bookmarks {
			if e, err := Get(b.EntityId); err != nil {
				log.Errorf("failed to get bookmarked event %s – %v\n", b.EntityId, err)
			} else {
				events[i] = e
			}
		}

		eventRatings := make(map[string]float32)
		for _, e := range events {
			eventRatings[e.Id] = 0
			if averages, err := reviews.GetAverages(e.Id); err == nil {
				if avg, ok := averages[reviews.KeyMainRating]; ok {
					eventRatings[e.Id] = avg
				}
			}
		}

		c.HTML(http.StatusOK, "bookmarks", gin.H{
			"events":       events,
			"eventRatings": eventRatings,
			"tplCtx":       c.MustGet(config.TemplateContextKey),
		})
	}
}

func getEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		semester := common.SemesterKeys[len(common.SemesterKeys)-1]
		if s := c.Request.URL.Query().Get("semester"); s != "" {
			semester = common.SemesterKey(s)
		}

		var bookmarked bool

		var user *users.User
		u, _ := c.Get(config.UserKey)
		if u != nil {
			user = u.(*users.User)
			if _, err := FindBookmark(user.Id, event.Id); err == nil {
				bookmarked = true
			}
		}

		var comms []*comments.Comment
		var userReview *reviews.Review
		var averageRatings map[string]float32

		if user != nil {
			comms, err = comments.Find(&comments.CommentQuery{
				EventIdEq: event.Id,
				UserIdEq:  user.Id,
				ActiveEq:  true,
			})

			userReview, err = reviews.Get(fmt.Sprintf("%s:%s", user.Id, event.Id))

			averageRatings, err = reviews.GetAverages(event.Id)

			if err != nil {
				c.Error(err).SetType(gin.ErrorTypePrivate)
				util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
				return
			}
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"event":          event,
			"bookmarked":     bookmarked,
			"comments":       comms,
			"userReview":     userReview,
			"averageRatings": averageRatings,
			"semesterQuery":  semester,
			"tplCtx":         c.MustGet(config.TemplateContextKey),
		})
	}
}

func apiSearchEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		values := c.Request.URL.Query()

		// TODO: Add ability to search by ID
		eventQuery := &EventQuery{
			NameLike: values.Get("q"),
			Limit:    10,
		}

		events, err := Find(eventQuery)
		if err != nil {
			c.Error(err)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{})
			return
		}

		eventVms := make([]*EventSearchResultItem, len(events))
		for i, e := range events {
			eventVms[i] = NewEventSearchResultItem(e)
		}

		c.JSON(http.StatusOK, eventVms)
	}
}

func apiPutBookmark(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		event, err := Get(c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
			return
		}

		bm, err := FindBookmark(user.Id, event.Id)
		if err != nil {
			if err := InsertBookmark(&Bookmark{
				UserId:   user.Id,
				EntityId: event.Id,
			}); err != nil {
				c.Error(err)
				c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
				return
			}

			c.Status(http.StatusCreated)
		} else {
			if err := DeleteBookmark(bm); err != nil {
				c.Error(err)
				c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
				return
			}

			c.Status(http.StatusNoContent)
		}
	}
}

func buildEventQuery(v url.Values) *EventQuery {
	var (
		limit  = config.Get().Misc.Pagesize
		offset = 0
	)

	if limitStr := v.Get("limit"); limitStr != "" {
		if limitInt, err := strconv.Atoi(limitStr); err == nil {
			limit = limitInt
		}
	}
	if offsetStr := v.Get("offset"); offsetStr != "" {
		if offsetInt, err := strconv.Atoi(offsetStr); err == nil {
			offset = offsetInt
		}
	}

	return &EventQuery{
		NameLike:     v.Get("name"),
		TypeEq:       v.Get("type"),
		LecturerIdEq: v.Get("lecturer_id"),
		SemesterEq:   v.Get("semester"),
		CategoryIn:   v["category"],
		Skip:         offset,
		Limit:        limit,
	}
}
