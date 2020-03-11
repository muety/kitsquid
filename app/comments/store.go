package comments

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"reflect"
	"time"
)

var (
	db            *bolthold.Store
	cfg           *config.Config
	commentsCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	commentsCache = cache.New(cfg.CacheDuration("comments", 30*time.Minute), cfg.CacheDuration("comments", 30*time.Minute)*2)

	setup()
	reindex()
}

func reindex() {
	r := func(name string) {
		if r := recover(); r != nil {
			log.Errorf("failed to reindex %s store\n", name)
		}
	}

	for _, t := range []interface{}{&Comment{}} {
		tn := reflect.TypeOf(t).String()
		log.Infof("reindexing %s", tn)
		func() {
			defer r(tn)
			db.ReIndex(t, nil)
		}()
	}
}

func setup() {}

func Get(id string) (*Comment, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if c, ok := commentsCache.Get(cacheKey); ok {
		return c.(*Comment), nil
	}

	var comment Comment

	if err := db.Get(id, &comment); err != nil {
		return &comment, err
	}

	commentsCache.SetDefault(cacheKey, &comment)
	return &comment, nil
}

func Find(query *CommentQuery) ([]*Comment, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if cc, ok := commentsCache.Get(cacheKey); ok {
		return cc.([]*Comment), nil
	}

	var foundComments []*Comment

	q := bolthold.
		Where("Id").Not().Eq("").Index("Id").
		And("Active").Eq(query.ActiveEq).Index("Active")

	if query != nil {
		if query.EventIdEq != "" {
			q.And("EventId").Eq(query.EventIdEq).Index("EventId")
		}
		if query.UserIdEq != "" {
			q.And("UserId").Eq(query.UserIdEq).Index("UserId")
		}
		if query.Skip > 0 {
			q.Skip(query.Skip)
		}
		if query.Limit > 0 {
			q.Limit(query.Limit)
		}
	}

	err := db.Find(&foundComments, q.SortBy("CreatedAt"))
	if err == nil {
		commentsCache.SetDefault(cacheKey, foundComments)
	}
	return foundComments, err
}

func Insert(comment *Comment, upsert bool) error {
	commentsCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}

	return f(comment.Id, comment)
}

func Delete(id string) error {
	commentsCache.Flush()
	return db.Delete(id, &Comment{})
}

func GetMaxIndexByEvent(eventId string) (count uint8, err error) {
	cacheKey := fmt.Sprintf("max:index:%s", eventId)
	if e, ok := commentsCache.Get(cacheKey); ok {
		return e.(uint8), err
	}

	q := bolthold.
		Where("EventId").Eq(eventId).Index("EventId")

	var result Comment

	aggResult, err := db.FindAggregate(&result, q)
	if err == nil && len(aggResult) == 1 && aggResult[0].Count() > 0 {
		aggResult[0].Max("Index", &result)
		count = result.Index
		commentsCache.SetDefault(cacheKey, count)
	}

	return count, err
}
