package common

import (
	"regexp"
)

/*
SemesterKey represents a semester identifier
*/
type SemesterKey string

/*
Genders represents a user's gender
*/
var Genders = []string{"male", "female", "human"}

/*
UserWhitelistItem is used to specify a set of requirements which a user has to fulfil in order to register
*/
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

/*
Validate checks whether the current item contains valid regexes
*/
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

/*
MailLocalPartRegex returns the regex to validate the everything before the @-sign of a user's email address
*/
func (u *UserWhitelistItem) MailLocalPartRegex() *regexp.Regexp {
	if u.localPartRegex == nil {
		u.localPartRegex = regexp.MustCompile(u.MailPrefixPattern)
	}
	return u.localPartRegex
}

/*
MailDomainRegex returns the regex to validate the everything after the @-sign of a user's email address
*/
func (u *UserWhitelistItem) MailDomainRegex() *regexp.Regexp {
	if u.localPartRegex == nil {
		u.localPartRegex = regexp.MustCompile(u.MailSuffixPattern)
	}
	return u.localPartRegex
}

/*
PasswordRegex returns the regex to validate a user's chosen password
*/
func (u *UserWhitelistItem) PasswordRegex() *regexp.Regexp {
	if u.passwordRegex == nil {
		u.passwordRegex = regexp.MustCompile(u.PasswordPattern)
	}
	return u.passwordRegex
}
