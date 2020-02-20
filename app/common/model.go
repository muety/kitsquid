package common

import (
	"regexp"
)

type SemesterKey string

const (
	SemesterWs1920 SemesterKey = "WS19/20"
	SemesterWs1718 SemesterKey = "WS17/18"
)

var SemesterKeys = []SemesterKey{SemesterWs1920, SemesterWs1718}
var Genders = []string{"male", "female", "human"}

type UserWhitelistItem struct {
	MailPrefixPattern string `yaml:"prefix-pattern"`
	MailPrefixDisplay string `yaml:"prefix-display"`
	MailSuffixPattern string `yaml:"suffix-pattern"`
	MailSuffixDisplay string `yaml:"suffix-display"`
	PasswordPattern   string `yaml:"password-pattern"`
	localPartRegex    *regexp.Regexp
	domainRegex       *regexp.Regexp
	passwordRegex     *regexp.Regexp
}

func (u *UserWhitelistItem) Validate() error {
	if _, err := regexp.Compile(u.MailPrefixPattern); err != nil {
		return err
	}
	if _, err := regexp.Compile(u.MailSuffixPattern); err != nil {
		return err
	}
	if _, err := regexp.Compile(u.PasswordPattern); err != nil {
		return err
	}
	return nil
}

func (u *UserWhitelistItem) MailLocalPartRegex() *regexp.Regexp {
	if u.localPartRegex == nil {
		u.localPartRegex = regexp.MustCompile(u.MailPrefixPattern)
	}
	return u.localPartRegex
}

func (u *UserWhitelistItem) MailDomainRegex() *regexp.Regexp {
	if u.localPartRegex == nil {
		u.localPartRegex = regexp.MustCompile(u.MailSuffixPattern)
	}
	return u.localPartRegex
}

func (u *UserWhitelistItem) PasswordRegex() *regexp.Regexp {
	if u.passwordRegex == nil {
		u.passwordRegex = regexp.MustCompile(u.PasswordPattern)
	}
	return u.passwordRegex
}
