package store

import (
	"github.com/n1try/kithub2/app/config"
	"github.com/timshannon/bolthold"
)

var db *bolthold.Store

func Init() {
	db = config.Db()
	initDefaultCaches()
}
