package events

import (
	"fmt"
)

type Event struct {
	Id          string `boltholdIndex:"Id"`
	Gguid       string
	Name        string `boltholdIndex:"Name"`
	Type        string `boltholdIndex:"Type"`
	Description string
	Categories  []string `boltholdSliceIndex:"Categories"`
	Links       []*Link
	Dates       []*EventDate
	Lecturers   []*Lecturer
	Semesters   []string `boltholdSliceIndex:"Semesters"`
}

func (l *Event) Link(baseUrl string) string {
	return fmt.Sprintf("%s/event.asp?gguid=%s", baseUrl, l.Gguid)
}

func (l Event) String() string {
	return fmt.Sprintf("[%s] %s (%s) @ %v by %s\n", l.Id, l.Name, l.Type, l.Categories, l.Lecturers)
}

func (l Lecturer) String() string {
	return l.Name
}

type EventDate struct {
	Date string
	Room string
}

type Lecturer struct {
	Gguid string
	Name  string `boltholdIndex:"Name"`
}

type Link struct {
	Name string
	Url  string
}

type EventQuery struct {
	NameLike     string
	TypeEq       string
	LecturerIdEq string
	SemesterEq   string
	CategoryIn   []string
}

type Bookmark struct {
	Id       uint64 `boltholdKey:"Id"`
	UserId   string `boltholdIndex:"UserId"`
	EntityId string `boltholdIndex:"EntityId"`
}
