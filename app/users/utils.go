package users

import (
	"bytes"
	"encoding/json"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/common"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/util"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const recaptchaApiUrl = "https://www.google.com/recaptcha/api/siteverify"

var client *http.Client

func getHttpClient() *http.Client {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Minute,
		}
	}
	return client
}

func NewUserValidator(cfg *config.Config, checkPw bool) UserValidator {
	return func(u *User) bool {
		if !util.ContainsString(u.Degree, cfg.University.Degrees) ||
			!util.ContainsString(u.Major, cfg.University.Majors) ||
			!util.ContainsString(u.Gender, common.Genders) {
			return false
		}

		return NewUserCredentialsValidator(cfg, checkPw)(u)
	}
}

func NewUserCredentialsValidator(cfg *config.Config, checkPw bool) UserCredentialsValidator {
	return func(u *User) bool {
		whitelist := cfg.Auth.Whitelist

		for _, w := range whitelist {
			if w.MailDomainRegex().Match([]byte(u.Id)) &&
				w.MailLocalPartRegex().Match([]byte(u.Id)) &&
				(!checkPw || w.PasswordRegex().Match([]byte(u.Password))) {
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

// Inplace!
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
		"recipient": u.Id,
		"sender":    cfg.Mail.From,
		"link":      cfg.ActivationLink(activationCode),
	}); err != nil {
		return err
	}

	log.Infof("sending confirmation mail to %s", u.Id)

	return util.SendMail(u.Id, &buf)
}

func ValidateRecaptcha(token, ip string) bool {
	form := url.Values{}
	form.Add("secret", cfg.Recaptcha.ClientSecret)
	form.Add("response", token)
	form.Add("remoteip", ip)

	req, _ := http.NewRequest(http.MethodPost, recaptchaApiUrl, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := getHttpClient().Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	var apiResponse recaptchaApiResponse
	if err := json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
		return false
	}

	return apiResponse.Success
}

func DeletedUser() *User {
	return &User{
		Id:        config.DeletedUserName,
		Password:  "",
		Active:    true,
		Admin:     false,
		Gender:    "–",
		Major:     "–",
		Degree:    "–",
		CreatedAt: time.Time{},
	}
}
