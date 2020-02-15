package config

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/common"
	"net/smtp"
	"strconv"
	"time"
)

type Config struct {
	Env  string `default:"development" env:"KITHUB_ENV"`
	Port int    `default:"8080" env:"KITHUB_PORT"`
	Addr string `default:"" env:"KITHUB_ADDR"`
	Url  string `env:"KITHUB_URL"`
	Tls  struct {
		KeyPath  string `default:"etc/key.pem" env:"KITHUB_TLS_KEY"`
		CertPath string `default:"etc/cert.pem" env:"KITHUB_TLS_CERT"`
	}
	Db struct {
		Path     string `default:"kithub.db" env:"KITHUB_DB_FILE"`
		Encoding string `default:"gob" env:"KITHUB_DB_ENCODING"`
	}
	Mail struct {
		From string `default:"noreply@kithub.eu" env:"KITHUB_MAIL_SENDER"`
		Smtp struct {
			Host     string `env:"SMTP_HOST"`
			Port     int    `default:"25" env:"SMTP_PORT"`
			User     string `env:"SMTP_USER"`
			Password string `env:"SMTP_PASSWORD"`
		}
	}
	Cache map[string]string
	Auth  struct {
		Salt    string `default:"0" env:"KITHUB_AUTH_SALT"`
		Session struct {
			Timeout string `default:"1d" env:"KITHUB_AUTH_SESSION_TIMEOUT"`
		}
		Admin struct {
			User     string `env:"KITHUB_ADMIN_USER"`
			Password string `env:"KITHUB_ADMIN_PASSWORD"`
		}
		Whitelist []common.UserWhitelistItem
	}
	University struct {
		Majors  []string
		Degrees []string
		Genders []string
	}
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
