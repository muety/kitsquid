package config

import (
	"encoding/json"
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/timshannon/bolthold"
	"strings"
)

// TODO: Use proper i18n
var Messages = map[string]string{
	"signup_success":          "Du hast dich erfolgreich registriert. Eine Bestätigungsmail ist auf dem Weg in dein Postfach",
	"logout_success":          "Du hast dich erfolgreich ausgeloggt",
	"activate_failure":        "Dein Account konnte nicht aktiviert werden. Bitte wende dich an den Admin",
	"activate_success":        "Dein Account ist aktiviert. Du kannst dich jetzt einloggen",
	"password_change_success": "Dein Password wurde aktualisiert",
}

const (
	SessionKey            = "kitsquid_session"
	UserKey               = "user"
	TemplateContextKey    = "tplCtx"
	RemoteIpKey           = "remoteIp"
	KitVvzBaseUrl         = "https://campus.kit.edu/sp/campus/all"
	FacultyIdx            = 0
	MaxEventSearchResults = 25
)

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
	dbOpts := &bolthold.Options{
		Encoder: bolthold.DefaultEncode,
		Decoder: bolthold.DefaultDecode,
	}

	if strings.ToLower(config.Db.Encoding) == "json" {
		dbOpts.Encoder = json.Marshal
		dbOpts.Decoder = json.Unmarshal
	}

	if _db, err := bolthold.Open(config.Db.Path, 0600, dbOpts); err != nil {
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
