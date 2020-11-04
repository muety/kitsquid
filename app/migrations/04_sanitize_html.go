package migrations

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/muety/kitsquid/app/events"
	"github.com/timshannon/bolthold"
	"time"
)

type migration4 struct{}

func (m migration4) Id() string {
	return "migration_04_sanitize_html"
}

func (m migration4) Run(store *bolthold.Store) error {
	htmlPolicy := bluemonday.StrictPolicy()
	htmlPolicy.AllowNoAttrs()
	htmlPolicy.AllowImages()
	htmlPolicy.AllowLists()
	htmlPolicy.AllowTables()
	htmlPolicy.AllowElements("b", "i", "strong", "p", "span", "br", "h1", "h2", "h3", "h4", "h5", "h6", "a", "section")
	htmlPolicy.AllowAttrs("style").OnElements("p", "span")
	htmlPolicy.AllowStyles("text-decoration").MatchingEnum("underline", "line-through").OnElements("p", "span")

	var results []*events.Event
	if err := store.Find(&results, bolthold.Where("Description").Ne("")); err != nil {
		return err
	}

	for _, r := range results {
		r.Description = htmlPolicy.Sanitize(r.Description)
	}

	return events.InsertMulti(results, true, false)
}

func (m migration4) PostRun(store *bolthold.Store) error {
	return store.Insert(m.Id(), PersistedMigration{
		Id:        m.Id(),
		CreatedAt: time.Now(),
	})
}

func (m migration4) HasRun(store *bolthold.Store) bool {
	result := &PersistedMigration{}
	if err := store.Get(m.Id(), result); err != nil || result.Id != m.Id() {
		return false
	}
	return true
}
