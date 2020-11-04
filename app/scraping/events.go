package scraping

import (
	"context"
	"fmt"
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	model "github.com/n1try/kitsquid/app/events"
	"github.com/n1try/kitsquid/app/util"
	"golang.org/x/sync/semaphore"
	"golang.org/x/text/language"
	"math"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type FetchEventsJob struct {
	Tguid string
	From  int
	To    int
}

type listEventFacultiesJob struct {
	Tguid string
}

type listEventCategoriesJob struct {
	Tguid string
	Gguid string
}

type listEventsJob struct {
	Tguid    string
	Gguid    string
	Semester string
}

type listEventFacultiesResult struct {
	Semester  string
	Faculties []*eventFaculty
}

type eventFaculty struct {
	Gguid string
	Name  string
}

type eventCategory struct {
	Gguid string
	Name  string
}

type EventScraper struct{}

func NewEventScraper() *EventScraper {
	return &EventScraper{}
}

func (l EventScraper) Schedule(job ScrapeJob, cronExp string) {
}

func (l EventScraper) Run(job ScrapeJob) (interface{}, error) {
	return job.process()
}

func (l FetchEventsJob) process() (interface{}, error) {
	var semester string
	var events = make([]*model.Event, 0)
	var categories = make([]*eventCategory, 0)
	var faculties = make([]*eventFaculty, 0)

	makeError := func(err error) ([]*model.Event, error) {
		return events, err
	}

	job1 := listEventFacultiesJob{Tguid: l.Tguid}
	result1, err := job1.process()
	if err != nil {
		return makeError(err)
	}
	semester = result1.(listEventFacultiesResult).Semester
	faculties = result1.(listEventFacultiesResult).Faculties

	from := int(math.Max(float64(l.From), 0))
	to := int(math.Min(float64(l.To), float64(len(faculties)-1)))

	log.V(2).Infof("starting to scrape events for tguid %s from %d to %d", l.Tguid, from, to)

	for _, faculty := range faculties[from:to] {
		job2 := listEventCategoriesJob{Tguid: l.Tguid, Gguid: faculty.Gguid}
		result2, err := job2.process()
		if err != nil {
			log.Errorf("failed to fetch categories – %v\n", err)
			continue
		}
		categories = append(categories, result2.([]*eventCategory)...)
	}

	ctx := context.TODO()
	mtx := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(maxWorkers))

	eventMap := make(map[string]*model.Event)
	addEvents := func(eventList []*model.Event) {
		for _, l := range eventList {
			if item, ok := eventMap[l.Id]; !ok {
				eventMap[l.Id] = l
			} else {
				// Merge categories
				newCategories := make([]string, 0)
				for _, cat := range l.Categories {
					if !util.ContainsString(cat, item.Categories) {
						newCategories = append(newCategories, cat)
					}
				}
				item.Categories = append(item.Categories, newCategories...)
			}
		}
	}

	for _, cat := range categories {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Errorf("failed to acquire semaphore while fetching events – %v\n", err)
			continue
		}

		go func(tguid, gguid, semester string) {
			defer sem.Release(1)
			job := listEventsJob{Tguid: tguid, Gguid: gguid, Semester: semester}
			result, err := job.process()
			if err != nil {
				log.Errorf("failed to fetch events – %v\n", err)
				return
			}

			mtx.Lock()
			addEvents(result.([]*model.Event))
			mtx.Unlock()
			log.Flush()
		}(l.Tguid, cat.Gguid, semester)
	}

	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Errorf("failed to acquire semaphore – %v\n", err)
	}

	i := 0
	events = make([]*model.Event, len(eventMap))
	for _, l := range eventMap {
		events[i] = l
		i++
	}

	return events, nil
}

func (l listEventFacultiesJob) process() (interface{}, error) {
	faculties := make([]*eventFaculty, 0)
	semester := ""

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)
	reTitle := regexp.MustCompile(`.+ \((.+)\)`)
	reSemester := regexp.MustCompile(`(SS|WS) (\d{2}\/?\d{2})`)

	u, _ := url.Parse(facultiesUrl)
	q := u.Query()
	q.Add("tguid", l.Tguid)
	q.Add("lang", language.German.String()) // TODO: make configurable
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listEventFacultiesJob] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	// Parse heading (e.g. "Vorlesungsverzeichnis (WS 19/20)")
	h1, err := htmlquery.Query(doc, "//h1[@class='pagetitle']")
	if err != nil {
		log.Errorf("failed to query title in faculties document for tguid %s\n", l.Tguid)
		return nil, err
	}

	matches := reTitle.FindStringSubmatch(htmlquery.InnerText(h1))
	if len(matches) == 2 {
		matches2 := reSemester.FindStringSubmatch(matches[1])
		if len(matches2) == 3 {
			if matches2[1] == "SS" {
				matches2[2] = matches2[2][2:]
			}
			semester = fmt.Sprintf("%s %s", matches2[1], matches2[2])
		} else {
			log.Errorf("failed to parse title for tguid for %s\n", l.Tguid)
		}
	} else {
		log.Errorf("failed to parse title for tguid for %s\n", l.Tguid)
	}

	as, err := htmlquery.QueryAll(doc, "//table[@id='tableVVZ']/tbody[@class='tablecontent']//a")
	if err != nil {
		log.Errorf("failed to query faculties document for tguid %s\n", l.Tguid)
		return nil, err
	}

	for _, a := range as {
		name := htmlquery.InnerText(a)
		href := htmlquery.SelectAttr(a, "href")
		matches := reGguid.FindStringSubmatch(href)
		if len(matches) == 2 {
			faculties = append(faculties, &eventFaculty{
				Name:  name,
				Gguid: matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return listEventFacultiesResult{
		Semester:  semester,
		Faculties: faculties,
	}, nil
}

func (l listEventCategoriesJob) process() (interface{}, error) {
	categories := make([]*eventCategory, 0)

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listEventCategoriesJob] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	as, err := htmlquery.QueryAll(doc, "//td[contains(@class, 'indented')]/a")
	if err != nil {
		log.Errorf("failed to query categories document for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	for _, a := range as {
		name := htmlquery.InnerText(a)
		href := htmlquery.SelectAttr(a, "href")

		matches := reGguid.FindStringSubmatch(href)
		if len(matches) == 2 {
			categories = append(categories, &eventCategory{
				Name:  name,
				Gguid: matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return categories, nil
}

func (l listEventsJob) process() (interface{}, error) {
	events := make([]*model.Event, 0)

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)
	reStripPagetitle := regexp.MustCompile(`.+: +(.+) +\(.+\)`)
	reStripBreadcrumbTitle := regexp.MustCompile(`[\d\.]*\d? ?(.+)`)

	nameReplacer := strings.NewReplacer("\"", "", "„", "", "“", "")

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listEventsJob] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	categories := make([]string, 0)
	titles := make([]string, 0)

	// Extract child category from page title
	var childCatFound bool
	h1, err := htmlquery.Query(doc, "//h1[@class='pagetitle']")
	if err != nil {
		log.Errorf("failed to query events document for title for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}
	if title := htmlquery.InnerText(h1); title != "" {
		matches := reStripPagetitle.FindStringSubmatch(strings.ReplaceAll(title, "\n", ""))
		if len(matches) != 2 {
			log.Errorf("failed to parse title for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		} else {
			titles = append(titles, strings.Trim(matches[1], " "))
			childCatFound = true
		}
	}

	// Extract parent categories from breadcrumbs
	as, err := htmlquery.QueryAll(doc, "//li[@class='breadcrumb-item']/a")
	if err != nil {
		log.Errorf("failed to query events document for breadcrumbs for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	for i, a := range as {
		if i == 0 {
			continue
		}
		title := htmlquery.SelectAttr(a, "title")
		if title != "" {
			titles = append(titles, title)
		}
	}

	// Quick hack to have the faculty be the first slice item
	if childCatFound && len(titles) >= 2 {
		tmp := titles[0]
		titles[0] = titles[1]
		titles[1] = tmp
	}

	// Strip titles (e.g. "1.2 Vorlesungen" -> "Vorlesungen")
	for _, title := range titles {
		matches := reStripBreadcrumbTitle.FindStringSubmatch(title)
		if len(matches) != 2 {
			log.Errorf("failed to parse title for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		} else {
			categories = append(categories, matches[1])
		}
	}

	trs, err := htmlquery.QueryAll(doc, "//table[@id='EVENTLIST']/tbody[@class='tablecontent']/tr")
	if err != nil {
		log.Errorf("failed to query events document for rows for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	var currentEvent *model.Event
	for i, tr := range trs {
		if htmlquery.SelectAttr(tr, "id") != "" {
			// Case 1: Event row

			currentEvent = &model.Event{Categories: categories, Semesters: []string{l.Semester}}
			reLecturerId := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

			tds, err := htmlquery.QueryAll(tr, "/td")
			if err != nil || len(tds) != 8 {
				log.Errorf("failed to parse event columns for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}

			// LV-Nr
			a, err := htmlquery.Query(tds[1], "/a")
			if err != nil {
				log.Errorf("failed to get event id for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}
			currentEvent.Id = htmlquery.InnerText(a)

			// Titel
			a, err = htmlquery.Query(tds[2], "/a")
			if err != nil {
				log.Errorf("failed to get event title for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}
			currentEvent.Name = strings.TrimSpace(htmlquery.InnerText(a))
			currentEvent.Name = nameReplacer.Replace(currentEvent.Name)

			// Gguid
			if href := htmlquery.SelectAttr(a, "href"); href != "" {
				matches := reGguid.FindStringSubmatch(href)
				if len(matches) == 2 {
					currentEvent.Gguid = matches[1]
				} else {
					log.Errorf("failed to find gguid for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
					continue
				}
			} else {
				log.Errorf("failed to find gguid for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}

			// Art
			a, err = htmlquery.Query(tds[4], "/a")
			if err != nil {
				log.Errorf("failed to get event type for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}
			currentEvent.Type = htmlquery.InnerText(a)

			// Dozenten
			lecturers := make([]*model.Lecturer, 0)
			as, err := htmlquery.QueryAll(tds[3], "/a")
			if err != nil {
				log.Errorf("failed to get lecturers for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}
			for _, a := range as {
				lecturer := &model.Lecturer{}
				lecturer.Name = strings.TrimSpace(htmlquery.InnerText(a))
				lecturer.Name = nameReplacer.Replace(lecturer.Name)

				if href := htmlquery.SelectAttr(a, "href"); href != "" {
					matches := reLecturerId.FindStringSubmatch(href)
					if len(matches) == 2 {
						lecturer.Gguid = matches[1]
					} else {
						log.Errorf("failed to find lecturer gguid for %s\n", href)
					}
				}

				if lecturer.Gguid == "" {
					break
				}

				lecturers = append(lecturers, lecturer)
			}

			currentEvent.Lecturers = lecturers
			events = append(events, currentEvent)
		} else {
			// Case 2: Date row
			if currentEvent == nil {
				log.Errorf("tried to parse dates without an active event for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid)
				continue
			}

			tds, err := htmlquery.QueryAll(tr, "/td[contains(@class, 'collapsible')]")
			if err != nil {
				log.Errorf("failed to get dates for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
				continue
			}

			for _, td := range tds {
				dateEl, err := htmlquery.Query(td, "/span[contains(@class, 'date')]")
				if err != nil {
					log.Errorf("failed to get date for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
					continue
				}

				roomEl, err := htmlquery.Query(td, "/a[contains(@class, 'room')]")
				if err != nil {
					log.Errorf("failed to get room for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
					continue
				}

				if currentEvent.Dates == nil {
					currentEvent.Dates = make([]*model.EventDate, 0)
				}

				if dateEl != nil && roomEl != nil {
					currentEvent.Dates = append(currentEvent.Dates, &model.EventDate{
						Date: htmlquery.InnerText(dateEl),
						Room: htmlquery.InnerText(roomEl),
					})
				}
			}
		}
	}

	return events, nil
}
