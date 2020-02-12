package users

import (
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"golang.org/x/crypto/bcrypt"
)

func Validate(u *User) bool {
	cfg := config.Get()
	whitelist := config.Get().Auth.Whitelist

	if !util.ContainsString(u.Degree, cfg.University.Degrees) ||
		!util.ContainsString(u.Major, cfg.University.Majors) ||
		!util.ContainsString(u.Gender, cfg.University.Genders) {
		return false
	}

	for _, w := range whitelist {
		if w.MailDomainRegex().Match([]byte(u.Id)) &&
			w.MailLocalPartRegex().Match([]byte(u.Id)) &&
			w.PasswordRegex().Match([]byte(u.Password)) {
			return true
		}
	}

	return false
}

func HashPassword(u *User) error {
	cfg := config.Get()
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password+cfg.Auth.Salt), bcrypt.DefaultCost)
	if err == nil {
		u.Password = string(bytes)
	}
	return err
}

func CheckPasswordHash(u *User, plainPassword string) bool {
	cfg := config.Get()
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainPassword+cfg.Auth.Salt))
	return err == nil
}
