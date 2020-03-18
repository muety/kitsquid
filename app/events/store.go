package events

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/util"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
	"reflect"
	"regexp"
	"time"
)

var (
	db             *bolthold.Store
	cfg            *config.Config
	eventsCache    *cache.Cache
	bookmarksCache *cache.Cache
	miscCache      *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	eventsCache = cache.New(cfg.CacheDuration("events", 30*time.Minute), cfg.CacheDuration("events", 30*time.Minute)*2)
	miscCache = cache.New(cfg.CacheDuration("misc", 30*time.Minute), cfg.CacheDuration("misc", 30*time.Minute)*2)
	bookmarksCache = cache.New(cfg.CacheDuration("bookmarks", 30*time.Minute), cfg.CacheDuration("bookmarks", 30*time.Minute)*2)

	setup()
	Reindex()
}

func Reindex() {
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

func FlushCaches() {
	log.Infoln("flushing events caches")
	eventsCache.Flush()
	miscCache.Flush()
	bookmarksCache.Flush()
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
	return Find(nil)
}

func Find(query *EventQuery) ([]*Event, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ee, ok := eventsCache.Get(cacheKey); ok {
		return ee.([]*Event), nil
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
			q.And("Semesters").Contains(query.SemesterEq).Index("Semesters")
		}
		if len(query.CategoryIn) > 0 {
			// Isn't there a better solution?!
			cats := make([]interface{}, 0)
			for _, c := range query.CategoryIn {
				if c != "" {
					cats = append(cats, c)
				}
			}
			if len(cats) > 0 {
				q.And("Categories").ContainsAny(cats...).Index("Categories")
			}
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
	f := db.Insert

	if upsert {
		if existing, err := Get(event.Id); err == nil {
			f = db.Upsert
			updateEvent(event, existing)
		}
	}

	eventsCache.Flush()
	miscCache.Flush()
	return f(event.Id, event)
}

func InsertMulti(events []*Event, upsert bool) error {
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

	eventsCache.Flush()
	miscCache.Flush()
	return tx.Commit()
}

func Delete(key string) error {
	defer FlushCaches()

	return db.Bolt().Update(func(tx *bbolt.Tx) error {
		if err := db.TxDelete(tx, key, &Event{}); err != nil {
			return err
		}
		if err := db.TxDeleteMatching(tx, &Bookmark{}, bolthold.Where("EntityId").Eq(key)); err != nil {
			return err
		}
		return nil
	})
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
	cacheKey := "get:faculties"
	if fl, ok := miscCache.Get(cacheKey); ok {
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

	miscCache.SetDefault(cacheKey, faculties)
	return faculties, nil
}

func CountFaculties() int {
	if fl, err := GetFaculties(); err == nil {
		return len(fl)
	}
	return 0
}

func GetCategories() ([]string, error) {
	cacheKey := "get:categories"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	categoryMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		for _, c := range l.Categories {
			if _, ok := categoryMap[c]; !ok {
				categoryMap[c] = true
			}
		}
	}

	var i int
	categories := make([]string, len(categoryMap))
	for k := range categoryMap {
		categories[i] = k
		i++
	}

	miscCache.SetDefault(cacheKey, categories)
	return categories, nil
}

func GetTypes() ([]string, error) {
	cacheKey := "get:types"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	typeMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		if _, ok := typeMap[l.Type]; !ok {
			typeMap[l.Type] = true
		}
	}

	var i int
	types := make([]string, len(typeMap))
	for k := range typeMap {
		types[i] = k
		i++
	}

	miscCache.SetDefault(cacheKey, types)
	return types, nil
}

func GetLecturers() ([]*Lecturer, error) {
	cacheKey := "get:lecturers"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]*Lecturer), nil
	}

	lecturerMap := make(map[string]*Lecturer)
	events, err := GetAll()
	if err != nil {
		return []*Lecturer{}, err
	}

	for _, l := range events {
		for _, le := range l.Lecturers {
			if _, ok := lecturerMap[le.Gguid]; !ok {
				lecturerMap[le.Gguid] = le
			}
		}
	}

	var i int
	lecturers := make([]*Lecturer, len(lecturerMap))
	for _, v := range lecturerMap {
		lecturers[i] = v
		i++
	}

	miscCache.SetDefault(cacheKey, lecturers)
	return lecturers, nil
}

func GetBookmark(key uint64) (*Bookmark, error) {
	var bookmark Bookmark
	if err := db.Get(key, &bookmark); err != nil {
		return nil, err
	}
	return &bookmark, nil
}

func GetAllBookmarks() ([]*Bookmark, error) {
	cacheKey := "get:bookmark:all"
	if bb, ok := eventsCache.Get(cacheKey); ok {
		return bb.([]*Bookmark), nil
	}

	var bookmarks []*Bookmark
	if err := db.Find(&bookmarks, &bolthold.Query{}); err != nil {
		return bookmarks, err
	}

	bookmarksCache.SetDefault(cacheKey, bookmarks)
	return bookmarks, nil
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

func FindBookmarks(userId string) ([]*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s", userId)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.([]*Bookmark), nil
	}

	var bookmarks []*Bookmark
	if err := db.Find(&bookmarks, bolthold.
		Where("UserId").
		Eq(userId).
		Index("UserId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, bookmarks)
	return bookmarks, nil
}

func InsertBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Insert(bolthold.NextSequence(), bookmark)
}

func DeleteBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Delete(bookmark.Id, bookmark)
}
