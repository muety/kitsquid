package events

import (
	log "github.com/golang/glog"
	"github.com/leandro-lugaresi/hub"
	"github.com/muety/kitsquid/app/config"
)

func dispatchEventMessage(name string) func(m *hub.Message) {
	switch name {
	case config.EventAccountDelete:
		return handleAccountDelete
	}
	return func(m *hub.Message) {}
}

func handleAccountDelete(m *hub.Message) {
	userId, _ := m.Fields["id"]
	userComments, err := FindComments(&CommentQuery{
		UserIdEq: userId.(string),
		ActiveEq: true,
	})
	if err != nil {
		log.Errorf("failed to process %s – %v\n", m.Name, err)
	}

	var successCount int
	for _, c := range userComments {
		c.UserId = config.DeletedUserName
		// TODO: Batch insert
		if err := InsertComment(c, true); err != nil {
			log.Errorf("failed to update comments %s – %v\n", c.Id, err)
		} else {
			successCount++
		}
	}

	log.Infof("successfully updated %d comments for deleted user %s\n", successCount, userId)
}
