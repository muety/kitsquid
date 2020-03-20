package reviews

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"math"
	"reflect"
	"time"
)

var (
	db           *bolthold.Store
	cfg          *config.Config
	reviewsCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	reviewsCache = cache.New(cfg.CacheDuration("reviews", 30*time.Minute), cfg.CacheDuration("reviews", 30*time.Minute)*2)

	setup()
	Reindex()
}

func Reindex() {
	r := func(name string) {
		if r := recover(); r != nil {
			log.Errorf("failed to reindex %s store\n", name)
		}
	}

	for _, t := range []interface{}{&Review{}} {
		tn := reflect.TypeOf(t).String()
		log.Infof("reindexing %s", tn)
		func() {
			defer r(tn)
			db.ReIndex(t, nil)
		}()
	}
}

func FlushCaches() {
	log.Infoln("flushing reviews caches")
	reviewsCache.Flush()
}

func setup() {}

func Get(id string) (*Review, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if c, ok := reviewsCache.Get(cacheKey); ok {
		return c.(*Review), nil
	}

	var review Review

	if err := db.Get(id, &review); err != nil {
		return &review, err
	}

	reviewsCache.SetDefault(cacheKey, &review)
	return &review, nil
}

func Find(query *ReviewQuery) ([]*Review, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if rr, ok := reviewsCache.Get(cacheKey); ok {
		return rr.([]*Review), nil
	}

	var foundReviews []*Review

	q := bolthold.
		Where("Id").Not().Eq("").Index("Id")

	if query != nil {
		if query.EventIdEq != "" {
			q.And("EventId").Eq(query.EventIdEq).Index("EventId")
		}
		if query.UserIdEq != "" {
			q.And("UserId").Eq(query.UserIdEq).Index("UserId")
		}
	}

	err := db.Find(&foundReviews, q)
	if err == nil {
		reviewsCache.SetDefault(cacheKey, foundReviews)
	}
	return foundReviews, err
}

func GetAll() ([]*Review, error) {
	return Find(&ReviewQuery{})
}

func Count() int {
	cacheKey := "count"
	if c, ok := reviewsCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetAll()
	if err != nil {
		return -1
	}

	reviewsCache.SetDefault(cacheKey, len(all))
	return len(all)
}

func GetAverages(eventId string) (map[string]float32, error) {
	cacheKey := fmt.Sprintf("avg:%s", eventId)
	if avg, ok := reviewsCache.Get(cacheKey); ok {
		return avg.(map[string]float32), nil
	}

	result := make(map[string]float32)
	count := make(map[string]float32)

	reviews, err := Find(&ReviewQuery{
		EventIdEq: eventId,
	})

	if err != nil {
		return result, err
	}

	for _, r := range reviews {
		for k, v := range r.Ratings {
			if _, ok := result[k]; !ok {
				result[k] = 0
			}
			if _, ok := count[k]; !ok {
				count[k] = 0
			}
			result[k] += float32(v)
			count[k] += 1
		}
	}

	for k, v := range result {
		result[k] = float32(math.Round(float64(v/count[k])*10) / 10)
	}

	reviewsCache.SetDefault(cacheKey, result)
	return result, nil
}

func Insert(review *Review, upsert bool) error {
	review.Id = fmt.Sprintf("%s:%s", review.UserId, review.EventId)

	f := db.Insert
	if upsert {
		if existing, err := Get(review.Id); err == nil {
			f = db.Upsert
			updateReview(review, existing)
		}
	}

	reviewsCache.Flush()
	return f(review.Id, review)
}

func Delete(key string) error {
	defer reviewsCache.Flush()
	return db.Delete(key, &Review{})
}

// Inplace
func updateReview(newReview, existingReview *Review) {
	for k, v := range existingReview.Ratings {
		if _, ok := newReview.Ratings[k]; !ok {
			newReview.Ratings[k] = v
		}
	}
}
