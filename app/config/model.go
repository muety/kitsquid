package config

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/common"
	"net/smtp"
	"strconv"
	"time"
)

type Config struct {
	Env  string `default:"development" env:"KITSQUID_ENV"`
	Port int    `default:"8080" env:"KITSQUID_PORT"`
	Addr string `default:"" env:"KITSQUID_ADDR"`
	Url  string `env:"KITSQUID_URL"`
	Tls  struct {
		Enable   bool   `default:"false" env:"KITSQUID_TLS"`
		KeyPath  string `default:"etc/key.pem" env:"KITSQUID_TLS_KEY"`
		CertPath string `default:"etc/cert.pem" env:"KITSQUID_TLS_CERT"`
	}
	Db struct {
		Path     string `default:"kitsquid.db" env:"KITSQUID_DB_FILE"`
		Encoding string `default:"gob" env:"KITSQUID_DB_ENCODING"`
	}
	Mail struct {
		From string `default:"noreply@kitsquid.eu" env:"KITSQUID_MAIL_SENDER"`
		Smtp struct {
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
		Admins    []string `env:"KITSQUID_AUTH_ADMIN_USERS"`
		Whitelist []common.UserWhitelistItem
	}
	Misc struct {
		Pagesize int
	}
	University struct {
		Majors               []string
		Degrees              []string
		WinterSemesterPrefix string `default:"WS"`
		SummerSemesterPrefix string `default:"SS"`
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

func (c *Config) SessionTimeout() time.Duration {
	if d, err := time.ParseDuration(c.Auth.Session.Timeout); err == nil {
		return d
	}
	return 0
}

func (c *Config) SmtpHost() string {
	return fmt.Sprintf("%s:%d", c.Mail.Smtp.Host, c.Mail.Smtp.Port)
}

func (c *Config) SmtpAuth() smtp.Auth {
	return smtp.PlainAuth("", c.Mail.Smtp.User, c.Mail.Smtp.Password, c.Mail.Smtp.Host)
}

func (c *Config) ActivationLink(token string) string {
	return fmt.Sprintf("%s/activate?token=%s", c.Url, token)
}

func (c *Config) IsDev() bool {
	return c.Env == "development"
}
