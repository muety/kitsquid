package util

import (
	"bytes"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"net/mail"
	"net/smtp"
	"text/template"
)

/*
SendMail uses SMTP to send a new e-mail to the given recipient address with the given content
*/
func SendMail(recipient string, content *bytes.Buffer) error {
	cfg := config.Get()

	if _, err := mail.ParseAddress(recipient); err != nil {
		return err
	}

	return smtp.SendMail(
		cfg.SMTPHost(),
		cfg.SMTPAuth(),
		cfg.Mail.From,
		[]string{recipient},
		content.Bytes())
}

/*
SendTestMail uses SMTP to send a new test e-mail for debugging purposes
*/
func SendTestMail(recipient string) error {
	cfg := config.Get()

	tpl, err := template.ParseFiles("app/views/mail/test.tpl.txt")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]string{
		"recipient": recipient,
		"sender":    cfg.Mail.From,
	}); err != nil {
		return err
	}

	log.Infof("sending test mail to %s", recipient)

	return SendMail(recipient, &buf)
}
