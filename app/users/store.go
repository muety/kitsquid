package users

import (
	"fmt"
	"github.com/n1try/kithub2/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"time"
)

var (
	db            *bolthold.Store
	cfg           *config.Config
	usersCache    *cache.Cache
	sessionsCache *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()

	db = store

	usersCache = cache.New(cfg.CacheDuration("users", 30*time.Minute), cfg.CacheDuration("users", 30*time.Minute)*2)
	sessionsCache = cache.New(cfg.CacheDuration("sessions", 30*time.Minute), cfg.CacheDuration("sessions", 30*time.Minute)*2)
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

func GetSession(token string) (*UserSession, error) {
	if s, ok := sessionsCache.Get(token); ok && time.Since(s.(*UserSession).LastSeen) < cfg.SessionTimeout() {
		return s.(*UserSession), nil
	}

	var sess UserSession
	if err := db.Get(token, &sess); err != nil {
		return nil, err
	}

	//sessionsCache.SetDefault(token, &sess)
	return &sess, nil
}

func InsertSession(sess *UserSession, upsert bool) error {
	f := db.Insert
	if upsert {
		f = db.Upsert
	}
	return f(sess.Token, sess)
}

func DeleteSession(sess *UserSession) error {
	sessionsCache.Delete(sess.Token)
	return db.Delete(sess.Token, sess)
}
