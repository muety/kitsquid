package scraping

import (
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/model"
	"net/url"
	"strings"
)

type FetchDetailsJob struct {
	Lectures []*model.Lecture
}

type LectureDetailsScraper struct{}

func NewLectureDetailsScraper() *LectureDetailsScraper {
	return &LectureDetailsScraper{}
}

func (l LectureDetailsScraper) Schedule(job ScrapeJob, cronExp string) {
}

func (l LectureDetailsScraper) Run(job ScrapeJob) (interface{}, error) {
	return job.process()
}

func (f FetchDetailsJob) process() (interface{}, error) {
	var updatedLectures = make([]*model.Lecture, len(f.Lectures))

	for i, l := range f.Lectures {
		u, _ := url.Parse(eventUrl)
		q := u.Query()
		q.Add("gguid", l.Gguid)
		u.RawQuery = q.Encode()

		log.V(2).Infof("[FetchDetailsJob] processing %s\n", u.String())
		doc, err := htmlquery.LoadURL(u.String())
		if err != nil {
			log.Errorf("failed to load %s\n", u.String())
			return nil, err
		}

		var sb strings.Builder

		ps, err := htmlquery.QueryAll(doc, "//div[@id='rwev_note']/p")
		if err != nil {
			log.Errorf("failed to query event document for description %s\n", l.Gguid)
			return nil, err
		}

		for _, el := range ps {
			sb.WriteString(htmlquery.OutputHTML(el, false))
		}

		newLecture := *l
		newLecture.Description = sb.String()
		updatedLectures[i] = &newLecture
	}

	return updatedLectures, nil
}
