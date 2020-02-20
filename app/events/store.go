package events

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"reflect"
	"regexp"
	"time"
)

var (
	db             *bolthold.Store
	cfg            *config.Config
	eventsCache    *cache.Cache
	facultiesCache *cache.Cache
	bookmarksCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	eventsCache = cache.New(cfg.CacheDuration("events", 30*time.Minute), cfg.CacheDuration("events", 30*time.Minute)*2)
	facultiesCache = cache.New(cfg.CacheDuration("faculties", 30*time.Minute), cfg.CacheDuration("faculties", 30*time.Minute)*2)
	bookmarksCache = cache.New(cfg.CacheDuration("bookmarks", 30*time.Minute), cfg.CacheDuration("bookmarks", 30*time.Minute)*2)

	setup()
	reindex()
}

func reindex() {
	r := func(name string) {
		if r := recover(); r != nil {
			log.Errorf("failed to reindex %s store\n", name)
		}
	}

	for _, t := range []interface{}{&Event{}, &Bookmark{}} {
		tn := reflect.TypeOf(t).String()
		log.Infof("reindexing %s", tn)
		func() {
			defer r(tn)
			db.ReIndex(t, nil)
		}()
	}
}

func setup() {}

func Get(id string) (*Event, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := eventsCache.Get(cacheKey); ok {
		return l.(*Event), nil
	}

	var event Event
	if err := db.Get(id, &event); err != nil {
		return nil, err
	}

	eventsCache.SetDefault(cacheKey, &event)
	return &event, nil
}

func GetAll() ([]*Event, error) {
	return FindAll(nil)
}

func FindAll(query *EventQuery) ([]*Event, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ll, ok := eventsCache.Get(cacheKey); ok {
		return ll.([]*Event), nil
	}

	var foundEvents []*Event

	q := bolthold.Where("Id").Not().Eq("").Index("Id")

	if query != nil {
		if query.NameLike != "" {
			if re, err := regexp.Compile("(?i)" + query.NameLike); err == nil {
				q.And("Name").RegExp(re)
			}
		}
		if query.TypeEq != "" {
			q.And("Type").Eq(query.TypeEq).Index("Type")
		}
		if query.LecturerIdEq != "" {
			q.And("Lecturers").MatchFunc(func(ra *bolthold.RecordAccess) (bool, error) {
				if field := ra.Field(); field != nil {
					for _, item := range field.([]*Lecturer) {
						if item.Gguid == query.LecturerIdEq {
							return true, nil
						}
					}
				}
				return false, nil
			})
		}
		if query.SemesterEq != "" {
			q.And("Semesters").ContainsAny(query.SemesterEq)
		}
		if len(query.CategoryIn) > 0 {
			// Isn't there a better solution?!
			cats := make([]interface{}, len(query.CategoryIn))
			for i, c := range query.CategoryIn {
				cats[i] = c
			}
			q.And("Categories").ContainsAny(cats...)
		}
		if query.Skip > 0 {
			q.Skip(query.Skip)
		}
		if query.Limit > 0 {
			q.Limit(query.Limit)
		}
	}

	err := db.Find(&foundEvents, q.SortBy("Name"))
	if err == nil {
		eventsCache.SetDefault(cacheKey, foundEvents)
	}
	return foundEvents, err
}

func Insert(event *Event, upsert bool) error {
	eventsCache.Flush()
	facultiesCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}

	if upsert {
		if existing, err := Get(event.Id); err == nil {
			updateEvent(event, existing)
		}
	}

	return f(event.Id, event)
}

func InsertMulti(events []*Event, upsert bool) error {
	eventsCache.Flush()
	facultiesCache.Flush()

	f := db.TxInsert
	if upsert {
		f = db.TxUpsert
	}

	tx, err := db.Bolt().Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range events {
		if upsert {
			if existing, err := Get(e.Id); err == nil {
				updateEvent(e, existing)
			}
		}

		if err := f(tx, e.Id, e); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Inplace
func updateEvent(newEvent, existingEvent *Event) {
	for _, c := range existingEvent.Semesters {
		if !util.ContainsString(c, newEvent.Semesters) {
			newEvent.Semesters = append(newEvent.Semesters, c)
		}
	}
}

func GetFaculties() ([]string, error) {
	cacheKey := "get:all"
	if fl, ok := facultiesCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	facultyMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		if len(l.Categories) > config.FacultyIdx {
			if _, ok := facultyMap[l.Categories[config.FacultyIdx]]; !ok {
				facultyMap[l.Categories[config.FacultyIdx]] = true
			}
		}
	}

	var i int
	faculties := make([]string, len(facultyMap))
	for k := range facultyMap {
		faculties[i] = k
		i++
	}

	facultiesCache.SetDefault("get:all", faculties)
	return faculties, nil
}

func CountFaculties() int {
	if fl, err := GetFaculties(); err == nil {
		return len(fl)
	}
	return 0
}

func FindBookmark(userId, entityId string) (*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s:%s", userId, entityId)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.(*Bookmark), nil
	}

	var bookmark Bookmark
	if err := db.FindOne(&bookmark, bolthold.
		Where("UserId").
		Eq(userId).
		Index("UserId").
		And("EntityId").
		Eq(entityId).
		Index("EntityId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, &bookmark)
	return &bookmark, nil
}

func InsertBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Insert(bolthold.NextSequence(), bookmark)
}

func DeleteBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Delete(bookmark.Id, bookmark)
}
