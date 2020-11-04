package admin

import (
	"github.com/muety/kitsquid/app/config"
	"github.com/timshannon/bolthold"
)

var (
	db  *bolthold.Store
	cfg *config.Config
)

func initStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store
}
