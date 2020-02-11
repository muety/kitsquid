package model

import (
	"regexp"
	"time"
)

type User struct {
	Id        string
	Active    bool
	Gender    string
	CreatedAt time.Time
}

type UserWhitelistItem struct {
	MailPrefixPattern string `yaml:"prefix-pattern"`
	MailPrefixDisplay string `yaml:"prefix-display"`
	MailSuffixPattern string `yaml:"suffix-pattern"`
	MailSuffixDisplay string `yaml:"suffix-display"`
}

func (u *UserWhitelistItem) Validate() error {
	if _, err := u.MailLocalPartRegex(); err != nil {
		return err
	}
	if _, err := u.MailDomainRegex(); err != nil {
		return err
	}
	return nil
}

func (u *UserWhitelistItem) MailLocalPartRegex() (*regexp.Regexp, error) {
	return regexp.Compile(u.MailPrefixPattern)
}

func (u *UserWhitelistItem) MailDomainRegex() (*regexp.Regexp, error) {
	return regexp.Compile(u.MailSuffixPattern)
}
