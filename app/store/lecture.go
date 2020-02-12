package store

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/model"
	"github.com/timshannon/bolthold"
	"regexp"
)

func GetLecture(id string) (*model.Lecture, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := lecturesCache.Get(cacheKey); ok {
		return l.(*model.Lecture), nil
	}

	var lecture model.Lecture
	if err := db.Get(id, &lecture); err != nil {
		return nil, err
	}

	lecturesCache.SetDefault(cacheKey, &lecture)
	return &lecture, nil
}

func GetLectures() ([]*model.Lecture, error) {
	return FindLectures(nil)
}

// TODO: Use indices!!!
func FindLectures(query *model.LectureQuery) ([]*model.Lecture, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ll, ok := lecturesCache.Get(cacheKey); ok {
		return ll.([]*model.Lecture), nil
	}

	var lectures []*model.Lecture

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

	err := db.Find(&lectures, q)
	if err == nil {
		lecturesCache.SetDefault(cacheKey, lectures)
	}
	return lectures, err
}

func InsertLecture(lecture *model.Lecture, upsert bool) error {
	lecturesCache.Flush()
	facultiesCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}
	return f(lecture.Id, lecture)
}

func InsertLectures(lectures []*model.Lecture, upsert bool) error {
	lecturesCache.Flush()
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

	for _, l := range lectures {
		if err := f(tx, l.Id, l); err != nil {
			return err
		}
	}

	return tx.Commit()
}
