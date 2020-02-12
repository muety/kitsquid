package store

import (
	"fmt"
	"github.com/n1try/kithub2/app/model"
	"github.com/timshannon/bolthold"
	"regexp"
)

func GetEvent(id string) (*model.Event, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := eventsCache.Get(cacheKey); ok {
		return l.(*model.Event), nil
	}

	var event model.Event
	if err := db.Get(id, &event); err != nil {
		return nil, err
	}

	eventsCache.SetDefault(cacheKey, &event)
	return &event, nil
}

func GetEvents() ([]*model.Event, error) {
	return FindEvents(nil)
}

// TODO: Use indices!!!
func FindEvents(query *model.EventQuery) ([]*model.Event, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ll, ok := eventsCache.Get(cacheKey); ok {
		return ll.([]*model.Event), nil
	}

	var events []*model.Event

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
					for _, item := range field.([]*model.Lecturer) {
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

	err := db.Find(&events, q)
	if err == nil {
		eventsCache.SetDefault(cacheKey, events)
	}
	return events, err
}

func InsertEvent(event *model.Event, upsert bool) error {
	eventsCache.Flush()
	facultiesCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}
	return f(event.Id, event)
}

func InsertEvents(events []*model.Event, upsert bool) error {
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
