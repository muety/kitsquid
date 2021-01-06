package config

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/muety/kitsquid/app/common"
	"net/smtp"
	"strconv"
	"time"
)

/*
Config contains all configurable and derived properties in this application
*/
type Config struct {
	Env        string `default:"development" env:"KITSQUID_ENV"`
	Version    string
	QuickStart bool   `default:"false" env:"KITSQUID_QUICK_START"`
	Port       int    `default:"8080" env:"KITSQUID_PORT"`
	Addr       string `default:"" env:"KITSQUID_ADDR"`
	URL        string `env:"KITSQUID_URL"`
	Rate       string `default:"60-M" env:"KITSQUID_RATE_LIMIT"`
	TLS        struct {
		Enable   bool   `default:"false" env:"KITSQUID_TLS"`
		KeyPath  string `default:"etc/key.pem" yaml:"key" env:"KITSQUID_TLS_KEY"`
		CertPath string `default:"etc/cert.pem" yaml:"cert" env:"KITSQUID_TLS_CERT"`
	}
	Db struct {
		Path     string `default:"kitsquid.db" env:"KITSQUID_DB_FILE"`
		Encoding string `default:"gob" env:"KITSQUID_DB_ENCODING"`
	}
	Mail struct {
		From string `default:"no-reply@kitsquid.de" env:"KITSQUID_MAIL_SENDER"`
		SMTP struct {
			Host     string `env:"SMTP_HOST"`
			Port     int    `default:"25" env:"SMTP_PORT"`
			User     string `env:"SMTP_USER"`
			Password string `env:"SMTP_PASSWORD"`
		}
	}
	Cache map[string]string
	Auth  struct {
		Salt    string `default:"0" env:"KITSQUID_AUTH_SALT"`
		Session struct {
			Timeout string `default:"1d" env:"KITSQUID_AUTH_SESSION_TIMEOUT"`
		}
		Admin struct {
			User     string `env:"KITSQUID_ADMIN_USER"`
			Password string `env:"KITSQUID_ADMIN_PASSWORD"`
			Gender   string `env:"KITSQUID_ADMIN_GENDER"`
			Major    string `env:"KITSQUID_ADMIN_MAJOR"`
			Degree   string `env:"KITSQUID_ADMIN_DEGREE"`
		}
		Whitelist []common.UserWhitelistItem
	}
	Recaptcha struct {
		ClientID     string `yaml:"client_id" env:"KITSQUAD_RECAPTCHA_ID"`
		ClientSecret string `yaml:"client_secret" env:"KITSQUAD_RECAPTCHA_SECRET"`
	}
	Misc struct {
		Pagesize int `default:"50"`
	}
	University struct {
		Majors               []string
		Degrees              []string
		WinterSemesterPrefix string `default:"WS"`
		SummerSemesterPrefix string `default:"SS"`
	}
}

/*
Validate checks whether the loaded config is valid
*/
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

/*
ListenAddr returns the address + port string for the web server to listen on
*/
func (c *Config) ListenAddr() string {
	return c.Addr + ":" + strconv.Itoa(c.Port)
}

/*
CacheDuration returns a duration corresponding to the TTL of the cache with the given key
*/
func (c *Config) CacheDuration(key string, defaultVal time.Duration) time.Duration {
	if ds, ok := c.Cache[key]; ok {
		if d, err := time.ParseDuration(ds); err == nil {
			return d
		}
		log.Errorf("failed to parse cache duration for key %s\n", key)
	}
	return defaultVal
}

/*
SessionTimeout returns a duration corresponding to a session's timeout
*/
func (c *Config) SessionTimeout() time.Duration {
	if d, err := time.ParseDuration(c.Auth.Session.Timeout); err == nil {
		return d
	}
	return 0
}

/*
SMTPHost returns the address + port string of the SMTP server to be used
*/
func (c *Config) SMTPHost() string {
	return fmt.Sprintf("%s:%d", c.Mail.SMTP.Host, c.Mail.SMTP.Port)
}

/*
SMTPAuth returns an SMTP authentication object
*/
func (c *Config) SMTPAuth() smtp.Auth {
	return smtp.PlainAuth("", c.Mail.SMTP.User, c.Mail.SMTP.Password, c.Mail.SMTP.Host)
}

/*
ActivationLink generates an activation link Url from a given activation token
*/
func (c *Config) ActivationLink(token string) string {
	return fmt.Sprintf("%s/activate?token=%s", c.URL, token)
}

/*
IsDev returns whether or not the application is supposed to run in development mode
*/
func (c *Config) IsDev() bool {
	return c.Env == "development"
}
