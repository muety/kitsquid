package scraping

import (
	"context"
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/model"
	"golang.org/x/sync/semaphore"
	"net/url"
	"strings"
	"sync"
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

	ctx := context.TODO()
	mtx := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(maxWorkers))

	for i, l := range f.Lectures {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Errorf("failed to acquire semaphore while fetching lecture details – %v\n", err)
			continue
		}

		go func(index int, gguid string, existingLecture *model.Lecture) {
			defer sem.Release(1)
			u, _ := url.Parse(eventUrl)
			q := u.Query()
			q.Add("gguid", gguid)
			u.RawQuery = q.Encode()

			log.V(2).Infof("[FetchDetailsJob] processing %s\n", u.String())
			doc, err := htmlquery.LoadURL(u.String())
			if err != nil {
				log.Errorf("failed to load %s\n", u.String())
				return
			}

			var sb strings.Builder

			ps, err := htmlquery.QueryAll(doc, "//div[@id='rwev_note']/p")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}

			for _, el := range ps {
				sb.WriteString(htmlquery.OutputHTML(el, false))
			}

			newLecture := *existingLecture
			newLecture.Description = sb.String()

			mtx.Lock()
			updatedLectures[index] = &newLecture
			mtx.Unlock()
			log.Flush()
		}(i, l.Gguid, l)
	}

	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Errorf("failed to acquire semaphore – %v\n", err)
	}

	return updatedLectures, nil
}
