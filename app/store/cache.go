package store

import (
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	cache "github.com/patrickmn/go-cache"
	"time"
)

var (
	caches         = map[string]*cache.Cache{}
	eventsCache    *cache.Cache
	facultiesCache *cache.Cache
	usersCache     *cache.Cache
)

func initDefaultCaches() {
	eventsCache = GetOrInitCache("events", false)
	facultiesCache = GetOrInitCache("faculties", false)
	usersCache = GetOrInitCache("users", false)
}

func GetOrInitCache(key string, force bool) *cache.Cache {
	if c, ok := caches[key]; ok && !force {
		return c
	}

	d := config.Get().CacheDuration(key, 30*time.Minute)
	c := cache.New(d, d*2)
	caches[key] = c

	log.Infof("initialized cache '%s' with timeout of %v", key, d)

	return c
}
