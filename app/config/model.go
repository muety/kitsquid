package config

import (
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/common"
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
		Path string `default:"kithub.db" env:"KITHUB_DB_FILE"`
	}
	Cache map[string]string
	Auth  struct {
		Salt    string `default:"0" env:"KITHUB_AUTH_SALT"`
		Session struct {
			Timeout string `default:"1d" env:"KITHUB_AUTH_SESSION_TIMEOUT"`
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

func (c *Config) IsDev() bool {
	return c.Env == "development"
}
