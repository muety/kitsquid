package events

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/n1try/kitsquid/app/config"
	"github.com/timshannon/bolthold"
)

/*
Init initializes a new events store
*/
func Init(store *bolthold.Store, eventBus *hub.Hub) {
	initStore(store)
	initSubscriptions(eventBus)
}

func initSubscriptions(eventBus *hub.Hub) {
	sub := eventBus.Subscribe(
		0,
		config.EventAccountDelete,
	)

	go func(s hub.Subscription) {
		for msg := range s.Receiver {
			dispatchEventMessage(msg.Name)(&msg)
		}
	}(sub)
}
