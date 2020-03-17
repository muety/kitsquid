package reviews

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/n1try/kitsquid/app/common/errors"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/users"
	"net/http"
)

func RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) {
}

func RegisterApiRoutes(router *gin.Engine, group *gin.RouterGroup) {
	group.PUT("/reviews", CheckUser(), apiPutReview(router))
}

func apiPutReview(r *gin.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		var review Review

		u, _ := c.Get(config.UserKey)
		user := u.(*users.User)

		if err := c.ShouldBindJSON(&review); err != nil || !ratingValid(&review) {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusBadRequest, errors.BadRequest{}).SetType(gin.ErrorTypePublic)
			return
		}

		review.UserId = user.Id

		if err := Insert(&review, true); err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		// TODO: Prevent posting reviews to non-existing events
		// TODO: Prevent posting non-existing rating keys

		updateUserReview, err := Get(fmt.Sprintf("%s:%s", review.UserId, review.EventId))
		updateAverageRatings, err := GetAverages(review.EventId)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		updateInfo := map[string]interface{}{
			"userRatings":    updateUserReview.Ratings,
			"averageRatings": updateAverageRatings,
		}

		c.JSON(http.StatusOK, updateInfo)
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
