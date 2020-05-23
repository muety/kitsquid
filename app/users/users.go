package users

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/timshannon/bolthold"
)

/*
Init initializes the user store
*/
func Init(store *bolthold.Store, eventBus *hub.Hub) {
	initStore(store)
}
