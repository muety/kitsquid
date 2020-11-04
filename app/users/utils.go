package users

import (
	"bytes"
	"encoding/json"
	log "github.com/golang/glog"
	"github.com/muety/kitsquid/app/common"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/util"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const recaptchaAPIURL = "https://www.google.com/recaptcha/api/siteverify"

var client *http.Client

func getHTTPClient() *http.Client {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Minute,
		}
	}
	return client
}

/*
NewUserValidator instantiates a new user validator
*/
func NewUserValidator(cfg *config.Config, checkPw bool) userValidator {
	return func(u *User) bool {
		if !util.ContainsString(u.Degree, cfg.University.Degrees) ||
			!util.ContainsString(u.Major, cfg.University.Majors) ||
			!util.ContainsString(u.Gender, common.Genders) {
			return false
		}

		return NewUserCredentialsValidator(cfg, checkPw)(u)
	}
}

/*
NewUserCredentialsValidator instantiates a new user credentials validator
*/
func NewUserCredentialsValidator(cfg *config.Config, checkPw bool) userCredentialsValidator {
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

/*
NewSessionValidator instantiates a new user credentials validator
*/
func NewSessionValidator(cfg *config.Config, resolveUser userResolver) userSessionValidator {
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

/*
HashPassword hashes the given user's plain text password in-place
*/
func HashPassword(u *User) error {
	// Inplace!
	cfg := config.Get()
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password+cfg.Auth.Salt), bcrypt.DefaultCost)
	if err == nil {
		u.Password = string(bytes)
	}
	return err
}

/*
CheckPasswordHash checks a given password string against the user's hashed password
*/
func CheckPasswordHash(u *User, plainPassword string) bool {
	cfg := config.Get()
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainPassword+cfg.Auth.Salt))
	return err == nil
}

/*
SendConfirmationMail sends a user registration confirmation mail with the given activation code for the given user
*/
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

/*
ValidateRecaptcha is used to validate a given reCaptcha token
*/
func ValidateRecaptcha(token, ip string) bool {
	form := url.Values{}
	form.Add("secret", cfg.Recaptcha.ClientSecret)
	form.Add("response", token)
	form.Add("remoteip", ip)

	req, _ := http.NewRequest(http.MethodPost, recaptchaAPIURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := getHTTPClient().Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	var apiResponse recaptchaAPIResponse
	if err := json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
		return false
	}

	return apiResponse.Success
}

/*
DeletedUser returns an empty, placeholder user object
*/
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
