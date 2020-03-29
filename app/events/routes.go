package events

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/common/errors"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/users"
	"github.com/n1try/kitsquid/app/util"
	uuid "github.com/satori/go.uuid"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/", getEvents(router))
	group.GET("/bookmarks", CheckUser(), getBookmarks(router))
	group.GET("/event/:id", getEvent(router))
	group.POST("/event/:id/comments", CheckUser(), postComment(router))
	group.POST("/event/:id/comments/delete", CheckUser(), deleteComment(router))
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.GET("/event/search", apiSearchEvents(router))
	group.PUT("/event/:id/bookmark", CheckUser(), apiPutBookmark(router))
	group.PUT("/event/:id/reviews", CheckUser(), apiPutReview(router))
}

func getEvent(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		semesters, _ := GetSemesters()
		semester := semesters[0]
		if s := c.Request.URL.Query().Get("semester"); s != "" && util.ContainsString(s, semesters) {
			semester = s
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

		var comms []*Comment
		var userReview *Review
		var userMap map[string]*users.User
		var averageRatings map[string]float32
		var countRatings int

		countRatings = FindCountReviews(&ReviewQuery{
			EventIdEq: event.Id,
		})

		if user != nil {
			comms, err = FindComments(&CommentQuery{
				EventIdEq: event.Id,
				ActiveEq:  true,
			})

			userMap = make(map[string]*users.User)
			for _, c := range comms {
				if u, err := users.Get(c.UserId); err == nil {
					userMap[u.Id] = u
				}
			}

			userReview, err = GetReview(fmt.Sprintf("%s:%s", user.Id, event.Id))
			averageRatings, err = GetReviewAverages(event.Id)

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
			"userMap":        userMap,
			"userReview":     userReview,
			"averageRatings": averageRatings,
			"countRatings":   countRatings,
			"semesterQuery":  semester,
			"tplCtx":         c.MustGet(config.TemplateContextKey),
		})
	}
}

func getEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		eventQuery := buildEventQuery(c.Request.URL.Query())
		if eventQuery.SemesterEq == "" {
			eventQuery.SemesterEq = MustGetCurrentSemester()
		}

		events, err := Find(eventQuery)
		if err != nil {
			c.Error(err)
			util.MakeError(c, "index", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		categories, _ := GetCategoriesAtIndex(config.FacultyIdx) // Top-level categories only
		types, _ := GetTypes()
		lecturers, _ := GetLecturers()
		semesters, _ := GetSemesters()

		c.HTML(http.StatusOK, "index", gin.H{
			"events":     events,
			"types":      types,
			"categories": categories,
			"lecturers":  lecturers,
			"semesters":  semesters,
			"limit":      eventQuery.Limit,
			"offset":     eventQuery.Skip,
			"tplCtx":     c.MustGet(config.TemplateContextKey),
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

		events := make([]*Event, 0)
		for _, b := range bookmarks {
			if e, err := Get(b.EntityId); err != nil {
				log.Errorf("failed to get bookmarked event %s – %v\n", b.EntityId, err)
			} else {
				events = append(events, e)
			}
		}

		c.HTML(http.StatusOK, "bookmarks", gin.H{
			"events": events,
			"tplCtx": c.MustGet(config.TemplateContextKey),
		})
	}
}

func postComment(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var comment Comment

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		if err := c.ShouldBind(&comment); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		text := template.HTMLEscapeString(comment.Text)
		text = strings.ReplaceAll(text, "\r\n", "<br>")
		text = strings.ReplaceAll(text, "\n", "<br>")

		comment.Id = uuid.NewV4().String()
		comment.Text = text
		comment.Active = true // TODO: Admin functionality to activate comments
		comment.CreatedAt = time.Now()
		comment.UserId = user.Id
		comment.EventId = event.Id
		if maxIdx, err := GetMaxCommentIndexByEvent(comment.EventId); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		} else {
			comment.Index = maxIdx + 1
		}

		if err := InsertComment(&comment, false); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		// TODO: Prevent posting comments to non-existing events
		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    fmt.Sprintf("/event/%s", comment.EventId),
		})
	}
}

func deleteComment(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var comment commentDelete

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		if err := c.ShouldBind(&comment); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusBadRequest, errors.BadRequest{}, nil)
			return
		}

		existing, err := GetComment(comment.Id)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusNotFound, errors.NotFound{}, nil)
			return
		}

		if existing.UserId != user.Id {
			util.MakeError(c, "event", http.StatusUnauthorized, errors.Unauthorized{}, nil)
			return
		}

		if err := DeleteComment(comment.Id); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			util.MakeError(c, "event", http.StatusInternalServerError, errors.Internal{}, nil)
			return
		}

		c.HTML(http.StatusOK, "redirect", gin.H{
			"tplCtx": c.MustGet(config.TemplateContextKey),
			"url":    fmt.Sprintf("/event/%s", existing.EventId),
		})
	}
}

func apiSearchEvents(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		values := c.Request.URL.Query()

		// TODO: Add ability to search by ID
		eventQuery := &EventQuery{
			NameLike: values.Get("q"),
			Limit:    config.MaxEventSearchResults,
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

func apiPutReview(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var review Review

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		event, err := Get(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
			return
		}

		review.EventId = event.Id
		review.UserId = user.Id
		review.CreatedAt = time.Now()

		if err := c.ShouldBindJSON(&review); err != nil || !ratingValid(&review) {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{}).SetType(gin.ErrorTypePublic)
			return
		}

		if err := InsertReview(&review, true); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		// TODO: Prevent posting reviews to non-existing events
		// TODO: Prevent posting non-existing rating keys
		updateUserReview, err1 := GetReview(fmt.Sprintf("%s:%s", review.UserId, review.EventId))
		updateAverageRatings, err2 := GetReviewAverages(review.EventId)

		if err1 != nil || err2 != nil {
			if err1 != nil {
				c.Error(err1).SetType(gin.ErrorTypePrivate)
			}
			if err2 != nil {
				c.Error(err2).SetType(gin.ErrorTypePrivate)
			}
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		if overall, ok := updateAverageRatings[config.OverallRatingKey]; ok {
			event.Rating = overall
			event.InverseRating = -overall

			if err := Insert(event, true, false); err != nil {
				c.Error(err).SetType(gin.ErrorTypePrivate)
				c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
				return
			}
		}

		updateInfo := map[string]interface{}{
			"userRatings":    updateUserReview.Ratings,
			"averageRatings": updateAverageRatings,
		}

		c.JSON(http.StatusOK, updateInfo)
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
		SortFields:   []string{"InverseRating", "Name"},
	}
}

func ratingValid(review *Review) bool {
	if review.EventId == "" {
		return false
	}
	if len(review.Ratings) < 1 {
		return false
	}
	for k, v := range review.Ratings {
		if k == "" || v < 1 || v > 5 {
			return false
		}
	}
	return true
}
