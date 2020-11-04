package events

import (
	"fmt"
	"github.com/muety/kitsquid/app/config"
	"strconv"
	"strings"
	"time"
)

/*
Event represents an events (lecture, tutorial, etc.) in this application
*/
type Event struct {
	Id            string `boltholdIndex:"Id"`
	Gguid         string
	Name          string `boltholdIndex:"Name"`
	Type          string `boltholdIndex:"Type"`
	Description   string
	Rating        float32  // for caching purposes only; actual rating is kept as reviews.Reviews
	InverseRating float32  `boltholdIndex:"InverseRating"`
	Categories    []string `boltholdSliceIndex:"Categories"`
	Links         []*Link
	Dates         []*EventDate
	Lecturers     []*Lecturer
	Semesters     []string `boltholdSliceIndex:"Semesters"`
}

/*
Link returns the Url to this event
*/
func (l *Event) Link(baseURL string) string {
	return fmt.Sprintf("%s/event.asp?gguid=%s", baseURL, l.Gguid)
}

/*
InWinter returns whether or not this event takes place in the winter term
*/
func (l *Event) InWinter() bool {
	cfg := config.Get()
	for _, s := range l.Semesters {
		if strings.HasPrefix(s, cfg.University.WinterSemesterPrefix) {
			return true
		}
	}
	return false
}

/*
InSummer returns whether or not this event takes place in the summer term
*/
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

/*
Semesters represents a sortable list of semester identifiers
*/
type Semesters []string

func (s Semesters) Len() int {
	return len(s)
}
func (s Semesters) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Semesters) Less(i, j int) bool {
	token1, year1 := s.split(s[i])
	token2, year2 := s.split(s[j])

	if year1 == year2 {
		if token1 == "WS" && token2 == "SS" {
			return true
		}
	}

	return year1 < year2
}

func (s Semesters) split(str string) (token string, year int) {
	if len(str) != 5 && len(str) != 8 {
		return token, year
	}

	if y, err := strconv.Atoi(str[len(str)-2:]); err == nil {
		year = y
	} else {
		return token, year
	}

	return str[:2], year
}

/*
EventDate represents date and location of an event
*/
type EventDate struct {
	Date string
	Room string
}

/*
Lecturer represents the lecturer of an event
*/
type Lecturer struct {
	Gguid string
	Name  string
}

/*
Link represents an external link of an event
*/
type Link struct {
	Name string
	Url  string
}

/*
EventQuery is used to query saved events
*/
type EventQuery struct {
	NameLike     string
	TypeEq       string
	LecturerIdEq string
	SemesterEq   string
	CategoryIn   []string
	Skip         int
	Limit        int
	SortFields   []string
}

/*
EventSearchResultItem represents an item in the result list of an event query
*/
type EventSearchResultItem struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Lecturers []string `json:"lecturers"`
}

/*
NewEventSearchResultItem instantiates a new EventSearchResultItem
*/
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

/*
Bookmark represents a bookmarked event
*/
type Bookmark struct {
	Id       uint64 `boltholdKey:"Id"`
	UserId   string `boltholdIndex:"UserId"`
	EntityId string `boltholdIndex:"EntityId"`
}

/*
Review represents an event review
*/
// TODO: View models!
type Review struct {
	Id        string           `json:"" boltholdIndex:"Id"`
	EventId   string           `json:"event_id" boltholdIndex:"EventId"`
	UserId    string           `json:"" boltholdIndex:"UserId"`
	Ratings   map[string]uint8 `json:"ratings"`
	CreatedAt time.Time        `json:"" boltholdIndex:"CreatedAt"`
}

/*
ReviewQuery is used to query for reviews
*/
type ReviewQuery struct {
	EventIdEq string
	UserIdEq  string
}

/*
Comment represents a comment for an event
*/
type Comment struct {
	Id        string    `form:"" boltholdIndex:"Id"`
	Index     uint8     `form:"" boltholdIndex:"Index"`
	EventId   string    `form:"event_id" boltholdIndex:"EventId"`
	UserId    string    `form:"" boltholdIndex:"UserId"`
	Active    bool      `form:"" boltholdIndex:"Active"`
	Text      string    `form:"text" binding:"required"`
	CreatedAt time.Time `form:""`
}

/*
CommentQuery is used to query for comments
*/
type CommentQuery struct {
	EventIdEq string
	UserIdEq  string
	ActiveEq  bool
	Skip      int
	Limit     int
}

type commentDelete struct {
	Id string `form:"id" binding:"required"`
}
