package events

import (
	"fmt"
	"github.com/n1try/kitsquid/app/config"
	"strings"
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

func (l *Event) InWinter() bool {
	cfg := config.Get()
	for _, s := range l.Semesters {
		if strings.HasPrefix(s, cfg.University.WinterSemesterPrefix) {
			return true
		}
	}
	return false
}

func (l *Event) InSummer() bool {
	cfg := config.Get()
	for _, s := range l.Semesters {
		if strings.HasPrefix(s, cfg.University.SummerSemesterPrefix) {
			return true
		}
	}
	return false
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
	Name  string
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
	Skip         int
	Limit        int
}

type EventSearchResultItem struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Lecturers []string `json:"lecturers"`
}

func NewEventSearchResultItem(event *Event) *EventSearchResultItem {
	lecturers := make([]string, len(event.Lecturers))
	for i, l := range event.Lecturers {
		lecturers[i] = l.Name
	}

	return &EventSearchResultItem{
		Id:        event.Id,
		Name:      event.Name,
		Type:      event.Type,
		Lecturers: lecturers,
	}
}

type Bookmark struct {
	Id       uint64 `boltholdKey:"Id"`
	UserId   string `boltholdIndex:"UserId"`
	EntityId string `boltholdIndex:"EntityId"`
}
