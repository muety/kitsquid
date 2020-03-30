package scraping

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	"github.com/microcosm-cc/bluemonday"
	"github.com/n1try/kitsquid/app/events"
	"golang.org/x/sync/semaphore"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	client *http.Client
)

type iliasResponse struct {
	Gguid string `json:"gguid" binding:"required"`
	Url   string `json:"url" binding:"required"`
}

type FetchDetailsJob struct {
	Events []*events.Event
}

type EventDetailsScraper struct{}

func NewEventDetailsScraper() *EventDetailsScraper {
	return &EventDetailsScraper{}
}

func (l EventDetailsScraper) Schedule(job ScrapeJob, cronExp string) {
}

func (l EventDetailsScraper) Run(job ScrapeJob) (interface{}, error) {
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
	return job.process()
}

func (f FetchDetailsJob) process() (interface{}, error) {
	var updatedEvents = make([]*events.Event, len(f.Events))

	ctx := context.TODO()
	mtx := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(maxWorkers))

	// Don't forget to update migration4 when changing the policy
	htmlPolicy := bluemonday.StrictPolicy()
	htmlPolicy.AllowNoAttrs()
	htmlPolicy.AllowImages()
	htmlPolicy.AllowLists()
	htmlPolicy.AllowTables()
	htmlPolicy.AllowElements("b", "i", "strong", "p", "span", "br", "h1", "h2", "h3", "h4", "h5", "h6", "a", "section")
	htmlPolicy.AllowAttrs("style").OnElements("p", "span")
	htmlPolicy.AllowStyles("text-decoration").MatchingEnum("underline", "line-through").OnElements("p", "span")

	for i, e := range f.Events {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Errorf("failed to acquire semaphore while fetching event details – %v\n", err)
			continue
		}

		go func(index int, gguid string, existingEvent *events.Event) {
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
				desc1   string
				desc2   string
				desc    string
				link    string
				extLink string
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
				desc = htmlPolicy.Sanitize(desc1)
			} else {
				desc = htmlPolicy.Sanitize(desc2)
			}

			// External Link
			extLinkEl, err := htmlquery.Query(doc, "//div[@id='rwev_link']/a")
			if err != nil {
				log.Errorf("failed to query event document for link %s\n", gguid)
				return
			}
			if extLinkEl != nil {
				extLink = htmlquery.SelectAttr(extLinkEl, "href")
			}

			// Shortlink
			linkEl, err := htmlquery.Query(doc, "//div[@id='shortlink']/input")
			if err != nil {
				log.Errorf("failed to query event document for shortlink %s\n", gguid)
				return
			}
			if linkEl != nil {
				link = htmlquery.SelectAttr(linkEl, "value")
			}

			newEvent := *existingEvent
			newEvent.Description = desc
			if existingEvent.Description != "" && newEvent.Description == "" {
				log.Warningf("WARNING! event %s had description before, but was removed", existingEvent.Gguid)
			}

			newEvent.Links = []*events.Link{{Name: "VVZ", Url: link}}
			if extLink != "" {
				newEvent.Links = append(newEvent.Links, &events.Link{Name: "Link", Url: extLink})
			}

			// Fetch ILIAS link
			u, _ = url.Parse(fmt.Sprintf(eventIliasUrl, newEvent.Gguid))
			log.V(2).Infof("[FetchDetailsJob] processing %s\n", u.String())
			req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
			if resp, err := client.Do(req); err != nil {
				log.Errorf("failed to fetch ILIAS event details for event %s\n", newEvent.Gguid)
			} else {
				defer resp.Body.Close()

				var iliasData iliasResponse
				dec := json.NewDecoder(resp.Body)
				if err := dec.Decode(&iliasData); err != nil {
					log.Errorf("failed to parse ILIAS response for event %s\n", newEvent.Gguid)
				}

				if _, err := url.Parse(iliasData.Url); err == nil &&
					(strings.HasPrefix(iliasData.Url, "http://") || strings.HasPrefix(iliasData.Url, "https://")) {
					newEvent.Links = append(newEvent.Links, &events.Link{Name: "ILIAS", Url: iliasData.Url})
				}
			}

			mtx.Lock()
			updatedEvents[index] = &newEvent
			mtx.Unlock()
			log.Flush()
		}(i, e.Gguid, e)
	}

	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Errorf("failed to acquire semaphore – %v\n", err)
	}

	return updatedEvents, nil
}
