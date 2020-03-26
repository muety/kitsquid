package util

import (
	"bytes"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"net/smtp"
	"text/template"
)

func SendMail(recipient string, content *bytes.Buffer) error {
	cfg := config.Get()

	return smtp.SendMail(
		cfg.SmtpHost(),
		cfg.SmtpAuth(),
		cfg.Mail.From,
		[]string{recipient},
		content.Bytes())
}

func SendTestMail(recipient string) error {
	tpl, err := template.ParseFiles("app/views/mail/test.tpl.txt")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]string{
		"recipient": recipient,
	}); err != nil {
		return err
	}

	log.Infof("sending test mail to %s", recipient)

	return SendMail(recipient, &buf)
}
