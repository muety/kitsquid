package store

import (
	"fmt"
	"github.com/n1try/kithub2/app/model"
)

func GetUser(id string) (*model.User, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := usersCache.Get(cacheKey); ok {
		return l.(*model.User), nil
	}

	var user model.User
	if err := db.Get(id, &user); err != nil {
		return nil, err
	}

	usersCache.SetDefault(cacheKey, &user)
	return &user, nil
}

func InsertUser(user *model.User, upsert bool) error {
	usersCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}
	return f(user.Id, user)
}
