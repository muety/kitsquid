package config

import (
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/timshannon/bolthold"
	"strconv"
	"time"
)

type Config struct {
	Env  string `default:"development" env:"KITHUB_ENV"`
	Port int    `default:"8080" env:"KITHUB_PORT"`
	Addr string `default:"" env:"KITHUB_ADDR"`
	Tls  struct {
		KeyPath  string `default:"etc/key.pem" env:"KITHUB_TLS_KEY"`
		CertPath string `default:"etc/cert.pem" env:"KITHUB_TLS_CERT"`
	}
	Db struct {
		Path string `field:"cache" default:"kithub.db" env:"KITHUB_DB_FILE"`
	}
	Cache map[string]string
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

	log.Infof("running in %s mode.\n", config.Env)
}

func Get() *Config {
	return config
}

func Db() *bolthold.Store {
	return db
}

func (c *Config) ListenAddr() string {
	return c.Addr + ":" + strconv.Itoa(c.Port)
}

func (c *Config) CacheDuration(key string, defaultVal time.Duration) time.Duration {
	if ds, ok := c.Cache[key]; ok {
		if d, err := time.ParseDuration(ds); err == nil {
			return d
		} else {
			log.Errorf("failed to parse cache duration for key %s\n", key)
		}
	}
	return defaultVal
}

func (c *Config) IsDev() bool {
	return c.Env == "development"
}
