package users

import (
	"fmt"
	"github.com/n1try/kithub2/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"time"
)

var (
	db         *bolthold.Store
	cfg        *config.Config
	usersCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()

	db = store

	usersCache = cache.New(cfg.CacheDuration("users", 30*time.Minute), cfg.CacheDuration("users", 30*time.Minute)*2)
}

func Get(id string) (*User, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := usersCache.Get(cacheKey); ok {
		return l.(*User), nil
	}

	var user User
	if err := db.Get(id, &user); err != nil {
		return nil, err
	}

	usersCache.SetDefault(cacheKey, &user)
	return &user, nil
}

func Insert(user *User, upsert bool) error {
	usersCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}
	return f(user.Id, user)
}
