package admin

import (
	"encoding/json"
	"github.com/n1try/kithub2/app/comments"
	"github.com/n1try/kithub2/app/events"
	"github.com/n1try/kithub2/app/reviews"
	"github.com/n1try/kithub2/app/users"
	"github.com/timshannon/bolthold"
	"strconv"
)

var (
	registry map[string]*registeredEntity
	entities []*registeredEntity
)

func Init(store *bolthold.Store) {
	InitStore(store)

	registry = make(map[string]*registeredEntity)
	entities = make([]*registeredEntity, 0)

	registerEntities()
}

func registerEntities() {
	registry["event"] = &registeredEntity{
		Name:     "Event",
		Instance: &events.Event{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return events.GetAll() },
			Get:  func(key string) (i interface{}, err error) { return events.Get(key) },
			Put: func(key, value string) error {
				var event events.Event
				if err := json.Unmarshal([]byte(value), &event); err != nil {
					return err
				}
				return events.Insert(&event, true)
			},
			Delete: events.Delete,
			Flush:  events.FlushCaches,
		},
	}

	registry["user"] = &registeredEntity{
		Name:     "User",
		Instance: &users.User{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return users.GetAll() },
			Get:  func(key string) (i interface{}, err error) { return users.Get(key) },
			Put: func(key, value string) error {
				var user users.User
				if err := json.Unmarshal([]byte(value), &user); err != nil {
					return err
				}
				return users.Insert(&user, true)
			},
			Delete: users.Delete,
			Flush:  users.FlushCaches,
		},
	}

	registry["usersession"] = &registeredEntity{
		Name:     "UserSession",
		Instance: &users.UserSession{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return users.GetAllSessions() },
			Get:  func(key string) (i interface{}, err error) { return users.GetSession(key) },
			Put: func(key, value string) error {
				var session users.UserSession
				if err := json.Unmarshal([]byte(value), &session); err != nil {
					return err
				}
				return users.InsertSession(&session, true)
			},
			Delete: func(key string) error { return users.DeleteSession(&users.UserSession{Token: key}) },
			Flush:  users.FlushCaches,
		},
	}

	registry["review"] = &registeredEntity{
		Name:     "Review",
		Instance: &reviews.Review{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return reviews.GetAll() },
			Get:  func(key string) (i interface{}, err error) { return reviews.Get(key) },
			Put: func(key, value string) error {
				var review reviews.Review
				if err := json.Unmarshal([]byte(value), &review); err != nil {
					return err
				}
				return reviews.Insert(&review, true)
			},
			Delete: reviews.Delete,
			Flush:  reviews.FlushCaches,
		},
	}

	registry["comment"] = &registeredEntity{
		Name:     "Comment",
		Instance: &comments.Comment{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return comments.GetAll() },
			Get:  func(key string) (i interface{}, err error) { return comments.Get(key) },
			Put: func(key, value string) error {
				var comment comments.Comment
				if err := json.Unmarshal([]byte(value), &comment); err != nil {
					return err
				}
				return comments.Insert(&comment, true)
			},
			Delete: comments.Delete,
			Flush:  comments.FlushCaches,
		},
	}

	registry["bookmark"] = &registeredEntity{
		Name:     "Bookmark",
		Instance: &events.Bookmark{},
		Resolvers: crudResolvers{
			List: func() (i interface{}, err error) { return events.GetAllBookmarks() },
			Get: func(key string) (i interface{}, err error) {
				intKey, err := strconv.Atoi(key)
				if err != nil {
					intKey = -1
				}
				return events.GetBookmark(uint64(intKey))
			},
			Put: func(key, value string) error {
				var bookmark events.Bookmark
				if err := json.Unmarshal([]byte(value), &bookmark); err != nil {
					return err
				}
				return events.InsertBookmark(&bookmark)
			},
			Delete: func(key string) error {
				intKey, err := strconv.Atoi(key)
				if err != nil {
					intKey = -1
				}
				return events.DeleteBookmark(&events.Bookmark{Id: uint64(intKey)})
			},
			Flush: events.FlushCaches,
		},
	}

	for _, v := range registry {
		entities = append(entities, v)
	}
}
