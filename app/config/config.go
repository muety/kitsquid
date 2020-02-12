package config

import (
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/n1try/kithub2/app/model"
	"github.com/timshannon/bolthold"
	"strconv"
	"time"
)

// TODO: Use proper i18n
const (
	StrAlertSignupSuccessful = "Du hast dich erfolgreich registriert. Eine Bestätigungsmail ist auf dem Weg in dein Postfach."
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
		Path string `default:"kithub.db" env:"KITHUB_DB_FILE"`
	}
	Cache map[string]string
	Auth  struct {
		Salt      string `default:"0" env:"KITHUB_AUTH_SALT"`
		Whitelist []model.UserWhitelistItem
	}
	University struct {
		Majors  []string
		Degrees []string
		Genders []string
	}
}

func (c *Config) Validate() error {
	if c.Auth.Whitelist != nil {
		for _, i := range c.Auth.Whitelist {
			if err := i.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
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
	if err := config.Validate(); err != nil {
		log.Fatalf("config is not valid – %v", err)
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
