package util

import "fmt"

func ComposeMail(recipient, subject, text string) []byte {
	return []byte(fmt.Sprintf("Content-Type: text/plain; charset=UTF-8\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s.\r\n", recipient, subject, text))
}
