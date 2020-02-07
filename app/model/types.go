package model

import (
	"fmt"
	"time"
)

type SemesterKey string

const (
	SemesterWs1819 SemesterKey = "WS18/19"
)

type Lecture struct {
	Id         string
	Gguid      string
	Name       string
	Type       string
	Categories []string
	Lecturers  []*Lecturer
}

type Lecturer struct {
	Gguid string
	Name  string
}

type Review struct {
	Id        string
	LectureId string
	Comment   string
	Ratings   map[string]uint8
}

type User struct {
	Id        string
	Active    bool
	Gender    string
	CreatedAt time.Time
}

type LectureQuery struct {
	NameLike     string
	TypeEq       string
	LecturerIdEq string
	CategoryIn   []string
}

func (l *Lecture) Link(baseUrl string) string {
	return fmt.Sprintf("%s/event.asp?gguid=%s", baseUrl, l.Gguid)
}

func (l Lecture) String() string {
	return fmt.Sprintf("[%s] %s (%s) @ %v by %s\n", l.Id, l.Name, l.Type, l.Categories, l.Lecturers)
}

func (l Lecturer) String() string {
	return l.Name
}
