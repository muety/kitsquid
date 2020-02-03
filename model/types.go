package model

import "fmt"

type SemesterKey string

const (
	SemesterWs1819 SemesterKey = "WS18/19"
)

type Faculty struct {
	Id   string
	Name string
}

type Category struct {
	Id   string
	Name string
}

type Lecturer struct {
	Id   string
	Name string
}

func (l Lecturer) String() string {
	return l.Name
}

type Lecture struct {
	Id         string
	Name       string
	Type       string
	FacultyIds []string
	Lecturers  []*Lecturer
}

func (l Lecture) String() string {
	return fmt.Sprintf("[%s] %s (%s) @ %v by %s\n", l.Id, l.Name, l.Type, l.FacultyIds, l.Lecturers)
}
