package events

import "github.com/timshannon/bolthold"

func Init(store *bolthold.Store) {
	InitStore(store)
}
