package util

import (
	"html/template"
	"strings"
)

func StrIndex(x int, v string) string {
	return string([]rune(v)[:1])
}

func StrRemove(html, needle string) string {
	return strings.ReplaceAll(html, needle, "")
}

func HtmlSafe(html string) template.HTML {
	return template.HTML(html)
}
