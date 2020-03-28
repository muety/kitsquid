package migrations

import (
	"github.com/n1try/kitsquid/app/events"
	"github.com/timshannon/bolthold"
	"strings"
	"time"
)

type migration3 struct{}

func (m migration3) Id() string {
	return "migration_03_names"
}

func (m migration3) Run(store *bolthold.Store) error {
	all, err := events.GetAll()
	if err != nil {
		return err
	}

	nameReplacer := strings.NewReplacer("\"", "", "„", "", "“", "")

	for _, e := range all {
		e.Name = nameReplacer.Replace(strings.TrimSpace(e.Name))

		for i, l := range e.Lecturers {
			e.Lecturers[i].Name = nameReplacer.Replace(strings.TrimSpace(l.Name))
		}
	}

	return events.InsertMulti(all, true, true)
}

func (m migration3) PostRun(store *bolthold.Store) error {
	return store.Insert(m.Id(), PersistedMigration{
		Id:        m.Id(),
		CreatedAt: time.Now(),
	})
}

func (m migration3) HasRun(store *bolthold.Store) bool {
	result := &PersistedMigration{}
	if err := store.Get(m.Id(), result); err != nil || result.Id != m.Id() {
		return false
	}
	return true
}
