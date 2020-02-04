package config

import (
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/timshannon/bolthold"
)

type Config struct {
	Db struct {
		Path string `default:"kithub.db" env:"KITHUB_DB_FILE"`
	}
}

var (
	config *Config
	db     *bolthold.Store
)

func Init() {
	if config != nil {
		return
	}

	// Load config
	config = &Config{}
	if err := configor.Load(config, "config.yml"); err != nil {
		log.Fatalf("failed to load config file — %v\n", err)
	}

	// Init database
	if _db, err := bolthold.Open(config.Db.Path, 0600, nil); err != nil {
		log.Fatalf("failed to open database — %v\n", err)
	} else {
		db = _db
	}
}

func Get() *Config {
	return config
}

func Db() *bolthold.Store {
	return db
}
