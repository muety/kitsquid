package events

import (
	"fmt"
	"github.com/n1try/kithub2/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"regexp"
	"time"
)

var (
	db             *bolthold.Store
	cfg            *config.Config
	eventsCache    *cache.Cache
	facultiesCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()

	db = store

	eventsCache = cache.New(cfg.CacheDuration("events", 30*time.Minute), cfg.CacheDuration("events", 30*time.Minute)*2)
	facultiesCache = cache.New(cfg.CacheDuration("faculties", 30*time.Minute), cfg.CacheDuration("faculties", 30*time.Minute)*2)
}

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

// TODO: Use indices!!!
func FindAll(query *EventQuery) ([]*Event, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ll, ok := eventsCache.Get(cacheKey); ok {
		return ll.([]*Event), nil
	}

	var foundEvents []*Event

	q := bolthold.Where("Id").Not().Eq("")

	if query != nil {
		if query.NameLike != "" {
			if re, err := regexp.Compile("(?i)" + query.NameLike); err == nil {
				q.And("Name").RegExp(re)
			}
		}
		if query.TypeEq != "" {
			q.And("Type").Eq(query.TypeEq)
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
		if len(query.CategoryIn) > 0 {
			q.And("Categories").ContainsAny(query.CategoryIn)
		}
	}

	err := db.Find(&foundEvents, q)
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

	for _, l := range events {
		if err := f(tx, l.Id, l); err != nil {
			return err
		}
	}

	return tx.Commit()
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
