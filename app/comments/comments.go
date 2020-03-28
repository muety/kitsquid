package comments

import "github.com/timshannon/bolthold"

// TODO: Refactor comments to be part of events package, analogously to reviews

func Init(store *bolthold.Store) {
	InitStore(store)
}
