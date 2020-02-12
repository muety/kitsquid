package util

import (
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/model"
	"golang.org/x/crypto/bcrypt"
)

func ValidateUser(u *model.User) bool {
	cfg := config.Get()
	whitelist := config.Get().Auth.Whitelist

	if !ContainsString(u.Degree, cfg.University.Degrees) ||
		!ContainsString(u.Major, cfg.University.Majors) ||
		!ContainsString(u.Gender, cfg.University.Genders) {
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

func HashPassword(u *model.User) error {
	cfg := config.Get()
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password+cfg.Auth.Salt), bcrypt.DefaultCost)
	if err == nil {
		u.Password = string(bytes)
	}
	return err
}

func CheckPasswordHash(u *model.User, plainPassword string) bool {
	cfg := config.Get()
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainPassword+cfg.Auth.Salt))
	return err == nil
}
