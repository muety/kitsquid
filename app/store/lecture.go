package store

import (
	"github.com/n1try/kithub2/app/model"
	"github.com/timshannon/bolthold"
)

func GetLectures() ([]*model.Lecture, error) {
	var lectures []*model.Lecture
	if err := db.Find(&lectures, &bolthold.Query{}); err != nil {
		return lectures, err
	}
	return lectures, nil
}

func InsertLecture(lecture *model.Lecture, upsert bool) error {
	f := db.Insert
	if upsert {
		f = db.Upsert
	}

	if err := f(lecture.Id, lecture); err != nil {
		return err
	}
	return nil
}

func InsertLectures(lectures []*model.Lecture, upsert bool) error {
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

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
