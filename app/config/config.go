package config

type Config struct{}

var config *Config

func Get() *Config {
	if config == nil {
		config = &Config{}
	}
	return config
}
