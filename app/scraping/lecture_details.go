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

			var (
				desc1 string
				desc2 string
				desc  string
				link  string
			)

			// Description from "Veranstaltungsdetails"
			noteEl, err := htmlquery.Query(doc, "//div[@id='rwev_note']")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}
			if noteEl != nil {
				desc1 = htmlquery.OutputHTML(noteEl, false)
			}

			// Description from "Weitere Informationen"
			var sb strings.Builder

			aimEl, err := htmlquery.Query(doc, "//div[@id='rwev_aim']")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}
			lcEl, err := htmlquery.Query(doc, "//div[@id='rwev_learningcontent']")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}
			prereqEl, err := htmlquery.Query(doc, "//div[@id='rwev_prereq']")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}
			workloadEl, err := htmlquery.Query(doc, "//div[@id='rwev_workload']")
			if err != nil {
				log.Errorf("failed to query event document for description %s\n", gguid)
				return
			}

			if aimEl != nil {
				sb.WriteString("<strong>Lernziele</strong><br>")
				sb.WriteString(htmlquery.OutputHTML(aimEl, false))
				sb.WriteString("<br><br>")
			}
			if lcEl != nil {
				sb.WriteString("<strong>Lehrinhalt</strong><br>")
				sb.WriteString(htmlquery.OutputHTML(lcEl, false))
				sb.WriteString("<br><br>")
			}
			if prereqEl != nil {
				sb.WriteString("<strong>Voraussetzungen</strong><br>")
				sb.WriteString(htmlquery.OutputHTML(prereqEl, false))
				sb.WriteString("<br><br>")
			}
			if workloadEl != nil {
				sb.WriteString("<strong>Arbeitsaufwand</strong><br>")
				sb.WriteString(htmlquery.OutputHTML(workloadEl, false))
				sb.WriteString("<br><br>")
			}
			desc2 = sb.String()

			if len(desc1) > len(desc2) {
				desc = desc1
			} else {
				desc = desc2
			}

			// Link
			linkEl, err := htmlquery.Query(doc, "//div[@id='rwev_link']/a")
			if err != nil {
				log.Errorf("failed to query event document for link %s\n", gguid)
				return
			}
			if linkEl != nil {
				link = htmlquery.SelectAttr(linkEl, "href")
			}

			newLecture := *existingLecture
			newLecture.Description = desc

			if newLecture.Links == nil {
				newLecture.Links = make([]string, 0)
			}
			if link != "" {
				newLecture.Links = append(newLecture.Links, link)
			}

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
