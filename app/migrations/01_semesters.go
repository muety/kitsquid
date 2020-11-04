package migrations

import (
	"fmt"
	"github.com/muety/kitsquid/app/events"
	"github.com/timshannon/bolthold"
	"strings"
	"time"
)

type migration1 struct{}

func (m migration1) Id() string {
	return "migration_01_semesters"
}

func (m migration1) Run(store *bolthold.Store) error {
	all, err := events.GetAll()
	if err != nil {
		return err
	}

	for _, e := range all {
		for i, s := range e.Semesters {
			if strings.HasPrefix(s, "WS") && !strings.Contains(s, " ") {
				// WS18/19
				e.Semesters[i] = fmt.Sprintf("%s %s", s[0:2], s[2:])
			} else if strings.HasPrefix(s, "SS") && !strings.Contains(s, " ") && len(s) == 4 {
				// SS20
				e.Semesters[i] = fmt.Sprintf("%s %s", s[0:2], s[2:])
			} else if strings.HasPrefix(s, "SS") && !strings.Contains(s, " ") && len(s) == 6 {
				// SS2020
				e.Semesters[i] = fmt.Sprintf("%s %s", s[0:2], s[4:])
			}
		}
	}

	if err := events.InsertMulti(all, true, true); err != nil {
		return err
	}

	return nil
}

func (m migration1) PostRun(store *bolthold.Store) error {
	return store.Insert(m.Id(), PersistedMigration{
		Id:        m.Id(),
		CreatedAt: time.Now(),
	})
}

func (m migration1) HasRun(store *bolthold.Store) bool {
	result := &PersistedMigration{}
	if err := store.Get(m.Id(), result); err != nil || result.Id != m.Id() {
		return false
	}
	return true
}
