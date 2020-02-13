package users

import (
	"bytes"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/util"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/smtp"
	"time"
)

func NewUserValidator(cfg *config.Config) UserValidator {
	return func(u *User) bool {
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
}

func NewSessionValidator(cfg *config.Config, resolveUser UserResolver) UserSessionValidator {
	return func(s *UserSession) bool {
		if user, err := resolveUser(s.UserId); err != nil || (!user.Active && !cfg.IsDev()) {
			return false
		}
		if (s.CreatedAt.Before(s.LastSeen) || s.CreatedAt.Equal(s.LastSeen)) &&
			time.Since(s.LastSeen) < cfg.SessionTimeout() {
			return true
		}
		return false
	}
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

func SendConfirmationMail(u *User, activationCode string) error {
	tpl, err := template.ParseFiles("app/views/mail/confirmation.tpl.txt")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]string{
		"Link": cfg.ActivationLink(activationCode),
	}); err != nil {
		return err
	}

	if err := smtp.SendMail(
		cfg.SmtpHost(),
		cfg.SmtpAuth(),
		cfg.Mail.From,
		[]string{u.Id},
		util.ComposeMail(u.Id, "Aktiviere Deinen KitHub Account", buf.String())); err != nil {
		return err
	}

	return nil
}
