package users

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
	"reflect"
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

	setup()
	reindex()
}

func reindex() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("failed to reindex user store")
		}
	}()

	for _, t := range []interface{}{&User{}, &Login{}} {
		tn := reflect.TypeOf(t).String()
		log.Infof("reindexing %s", tn)
		if err := db.ReIndex(t, nil); err != nil {
			log.Errorf("failed to reindex %s – %v\n", tn, err)
		}
	}
}

func setup() {
	if cfg.Auth.Admin.User != "" {
		if _, err := Get(cfg.Auth.Admin.User); err != nil {
			log.Infof("creating admin user %s", cfg.Auth.Admin.User)
			admin := &User{
				Id:        cfg.Auth.Admin.User,
				Password:  cfg.Auth.Admin.Password,
				Active:    true,
				Gender:    "",
				Major:     "",
				Degree:    "",
				CreatedAt: time.Now(),
			}

			if err := HashPassword(admin); err != nil {
				log.Errorf("failed to hash admin password – %v\n", err)
				return
			}

			if err := Insert(admin, false); err != nil {
				log.Errorf("failed to create admin user – %v\n", err)
				return
			}
		}
	}
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

	sessionsCache.SetDefault(token, &sess)
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

func GetToken(token string) (string, error) {
	var userId string
	err := db.Bolt().View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte("token")).Get([]byte(token))
		if v == nil {
			userId = ""
		}
		userId = string(v)
		return nil
	})
	return userId, err
}

func InsertToken(token, userId string) error {
	return db.Bolt().Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("token"))
		if err != nil {
			return err
		}
		return b.Put([]byte(token), []byte(userId))
	})
}

func DeleteToken(token string) error {
	return db.Bolt().Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("token"))
		if err != nil {
			return err
		}
		return b.Delete([]byte(token))
	})
}
