package util

import "golang.org/x/net/html"

func GetHTMLAttrValue(key string, attrs []html.Attribute) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
