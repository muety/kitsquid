package config

import (
	"encoding/json"
	log "github.com/golang/glog"
	"github.com/jinzhu/configor"
	"github.com/leandro-lugaresi/hub"
	"github.com/timshannon/bolthold"
	"io/ioutil"
	"os"
	"strings"
)

/*
Messages is a set of messages displayed to the user
*/
// TODO: Use proper i18n
var Messages = map[string]string{
	"signup_success":         "Du hast dich erfolgreich registriert. Eine Bestätigungsmail ist auf dem Weg in dein Postfach",
	"logout_success":         "Du hast dich erfolgreich ausgeloggt",
	"activate_failure":       "Dein Account konnte nicht aktiviert werden. Bitte wende dich an den Admin",
	"activate_success":       "Dein Account ist aktiviert. Du kannst dich jetzt einloggen",
	"account_change_success": "Deine Angaben wurden aktualisiert",
	"account_delete_success": "Dein Account wurde gelöscht",
}

const (
	SessionKey            = "kitsquid_session"
	UserKey               = "user"
	TemplateContextKey    = "tplCtx"
	RemoteIPKey           = "remoteIp"
	KitVvzBaseURL         = "https://campus.kit.edu/sp/campus/all"
	FacultyIdx            = 0
	OverallRatingKey      = "overall"
	DeletedUserName       = "(gelöschter Benutzer)"
	MaxEventSearchResults = 25
	EventAccountDelete    = "account.delete"
)

var (
	config   *Config
	db       *bolthold.Store
	eventBus *hub.Hub
)

/*
Init loads a config and performs all related initializations
*/
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

	// Read version
	config.Version = readVersion()

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

	// Init event bus
	eventBus = hub.New()

	log.Infof("running in %s mode.\n", config.Env)
}

func Get() *Config {
	return config
}

func Db() *bolthold.Store {
	return db
}

func EventBus() *hub.Hub {
	return eventBus
}

func readVersion() string {
	file, err := os.Open("version.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	return string(bytes)
}
