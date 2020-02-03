package scraping

import (
	"github.com/n1try/kithub2/model"
	"golang.org/x/text/language"
)

type LectureScraper interface {
	FetchLectures(semester model.SemesterKey, lang language.Tag) []model.Lecture
}
