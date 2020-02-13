package config

import (
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/timshannon/bolthold"
)

// TODO: Use proper i18n
var Messages = map[string]string{
	"signup_success": "Du hast dich erfolgreich registriert. Eine Bestätigungsmail ist auf dem Weg in dein Postfach",
	"logout_success": "Du hast dich erfolgreich ausgeloggt",
}

const (
	SessionKey         = "kithub_session"
	UserKey            = "user"
	TemplateContextKey = "tplCtx"
	KitVvzBaseUrl      = "https://campus.kit.edu/live-stud/campus/all"
	FacultyIdx         = 0
)

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
