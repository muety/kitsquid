package admin

import (
	"github.com/n1try/kithub2/app/config"
	"github.com/timshannon/bolthold"
)

var (
	db  *bolthold.Store
	cfg *config.Config
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store
}
