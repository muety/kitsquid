package model

type Review struct {
	Id        string
	LectureId string
	Comment   string
	Ratings   map[string]uint8
}
