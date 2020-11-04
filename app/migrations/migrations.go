package migrations

import (
	log "github.com/golang/glog"
	"github.com/muety/kitsquid/app/config"
	"github.com/timshannon/bolthold"
	"time"
)

type Migration interface {
	Id() string
	Run(store *bolthold.Store) error
	PostRun(store *bolthold.Store) error
	HasRun(store *bolthold.Store) bool
}

type PersistedMigration struct {
	Id        string `boltholdkey:"Id"`
	CreatedAt time.Time
}

var (
	registeredMigrations []Migration
)

func init() {
	registeredMigrations = []Migration{
		migration1{},
		migration2{},
		migration3{},
		migration4{},
	}
}

func RunAll() {
	db := config.Db()
	r := func(name string) {
		if r := recover(); r != nil {
			log.Errorf("failed to run migration %s\n", name)
		}
	}

	log.Infoln("starting database migrations")

	for _, m := range registeredMigrations {
		if !m.HasRun(db) {
			log.Infof("running migration %s\n", m.Id())

			func() {
				defer r(m.Id())
				if err := m.Run(db); err != nil {
					log.Errorf("failed to run migration %s – %v\n", m.Id(), err)
				}
				if err := m.PostRun(db); err != nil {
					log.Errorf("failed to run post migration hook for %s – %v\n", m.Id(), err)
				}
			}()
		}
	}

	log.Infoln("finished database migrations")
}
