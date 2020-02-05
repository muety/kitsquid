package model

import (
	"fmt"
)

type SemesterKey string

const (
	SemesterWs1819 SemesterKey = "WS18/19"
)

type Lecturer struct {
	Gguid string
	Name  string
}

func (l Lecturer) String() string {
	return l.Name
}

type Lecture struct {
	Id         string
	Gguid      string
	Name       string
	Type       string
	Categories []string
	Lecturers  []*Lecturer
}

func (l *Lecture) Link(baseUrl string) string {
	return fmt.Sprintf("%s/event.asp?gguid=%s", baseUrl, l.Gguid)
}

func (l Lecture) String() string {
	return fmt.Sprintf("[%s] %s (%s) @ %v by %s\n", l.Id, l.Name, l.Type, l.Categories, l.Lecturers)
}
