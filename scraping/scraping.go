package scraping

import (
	"context"
	"errors"
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/model"
	"github.com/n1try/kithub2/util"
	"golang.org/x/sync/semaphore"
	"golang.org/x/text/language"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

const (
	baseUrl      = model.KitVvzBaseUrl
	mainUrl      = baseUrl + "/field.asp"
	facultiesUrl = baseUrl + "/fields.asp?group=Vorlesungsverzeichnis"
	maxWorkers   = 6
)

// TODO: Move to external config or so
var tguids = map[model.SemesterKey]string{
	model.SemesterWs1819: "0x4CB7204338AE4F67A58AFCE6C29D1488",
}

type ScrapeJob interface {
	process() (interface{}, error)
}

type FetchLecturesJob struct {
	Semester model.SemesterKey
}

type listLectureFacultiesJob struct {
	Tguid string
}

type listLectureCategoriesJob struct {
	Tguid string
	Gguid string
}

type listLecturesJob struct {
	Tguid       string
	Gguid       string
	ParentGguid string
}

type lectureFaculty struct {
	Gguid string
	Name  string
}

type lectureCategory struct {
	Gguid string
	Name  string
}

type LectureScraper struct{}

func NewLectureScraper() *LectureScraper {
	return &LectureScraper{}
}

func (l *LectureScraper) Schedule(job ScrapeJob, cronExp string) {
}

func (l *LectureScraper) Run(job ScrapeJob) (interface{}, error) {
	return job.process()
}

func (l FetchLecturesJob) process() (interface{}, error) {
	var lectures = make([]*model.Lecture, 0)
	var categories = make([]*lectureCategory, 0)
	var faculties = make([]*lectureFaculty, 0)

	makeError := func(err error) ([]*model.Lecture, error) {
		return lectures, err
	}

	if _, ok := tguids[l.Semester]; !ok {
		return makeError(errors.New("unknown semester key"))
	}
	tguid := tguids[l.Semester]

	job1 := listLectureFacultiesJob{Tguid: tguid}
	result1, err := job1.process()
	if err != nil {
		return makeError(err)
	}
	faculties = result1.([]*lectureFaculty)[6:7]

	facultyByCategory := make(map[string]string)
	for _, faculty := range faculties {
		job2 := listLectureCategoriesJob{Tguid: tguid, Gguid: faculty.Gguid}
		result2, err := job2.process()
		if err != nil {
			log.Errorf("failed to fetch categories – %v\n", err)
			continue
		}
		categories = append(categories, result2.([]*lectureCategory)...)

		for _, cat := range result2.([]*lectureCategory) {
			facultyByCategory[cat.Gguid] = faculty.Gguid
		}
	}

	ctx := context.TODO()
	mtx := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(maxWorkers))

	lectureMap := make(map[string]*model.Lecture)
	addLectures := func(lectureList []*model.Lecture) {
		for _, l := range lectureList {
			if item, ok := lectureMap[l.Id]; !ok {
				lectureMap[l.Id] = l
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

	for _, cat := range categories[:1] {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Errorf("failed to acquire semaphore while fetching lectures – %v\n", err)
			continue
		}

		catId := cat.Gguid

		go func() {
			defer sem.Release(1)
			job := listLecturesJob{Tguid: tguid, Gguid: catId, ParentGguid: facultyByCategory[catId]}
			result, err := job.process()
			if err != nil {
				log.Errorf("failed to fetch lectures – %v\n", err)
				return
			}

			mtx.Lock()
			addLectures(result.([]*model.Lecture))
			mtx.Unlock()
			log.Flush()
		}()
	}

	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Errorf("failed to acquire semaphore – %v\n", err)
	}

	i := 0
	lectures = make([]*model.Lecture, len(lectureMap))
	for _, l := range lectureMap {
		lectures[i] = l
		i++
	}

	return lectures, nil
}

func (l listLectureFacultiesJob) process() (interface{}, error) {
	faculties := make([]*lectureFaculty, 0)

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

	u, _ := url.Parse(facultiesUrl)
	q := u.Query()
	q.Add("tguid", l.Tguid)
	q.Add("lang", language.German.String()) // TODO: make configurable
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listLectureFacultiesJob] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	as, err := htmlquery.QueryAll(doc, "//table[@id='tableVVZ']/tbody[@class='tablecontent']//a")
	if err != nil {
		log.Errorf("failed to query faculties document for tguid %s\n", l.Tguid)
		return nil, err
	}

	for _, a := range as {
		name := htmlquery.InnerText(a)
		href := util.GetHTMLAttrValue("href", a.Attr)
		matches := reGguid.FindStringSubmatch(href)
		if len(matches) == 2 {
			faculties = append(faculties, &lectureFaculty{
				Name:  name,
				Gguid: matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return faculties, nil
}

func (l listLectureCategoriesJob) process() (interface{}, error) {
	categories := make([]*lectureCategory, 0)

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listLectureCategoriesJob] processing %s\n", u.String())
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
		href := util.GetHTMLAttrValue("href", a.Attr)

		matches := reGguid.FindStringSubmatch(href)
		if len(matches) == 2 {
			categories = append(categories, &lectureCategory{
				Name:  name,
				Gguid: matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return categories, nil
}

func (l listLecturesJob) process() (interface{}, error) {
	lectures := make([]*model.Lecture, 0)

	reGguid := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)
	reStripPagetitle := regexp.MustCompile(`.+: +(.+) +\(.+\)`)
	reStripBreadcrumbTitle := regexp.MustCompile(`[\d\.]*\d? ?(.+)`)

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[listLecturesJob] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	categories := make([]string, 0)
	titles := make([]string, 0)

	// Extract child category from page title
	h1, err := htmlquery.Query(doc, "//h1[@class='pagetitle']")
	if err != nil {
		log.Errorf("failed to query lectures document for title for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}
	if title := htmlquery.InnerText(h1); title != "" {
		matches := reStripPagetitle.FindStringSubmatch(strings.ReplaceAll(title, "\n", ""))
		if len(matches) != 2 {
			log.Errorf("failed to parse title for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		} else {
			titles = append(titles, strings.Trim(matches[1], " "))
		}
	}

	// Extract parent categories from breadcrumbs
	as, err := htmlquery.QueryAll(doc, "//li[@class='breadcrumb-item']/a")
	if err != nil {
		log.Errorf("failed to query lectures document for breadcrumbs for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	for i, a := range as {
		if i == 0 {
			continue
		}
		title := util.GetHTMLAttrValue("title", a.Attr)
		if title != "" {
			titles = append(titles, title)
		}
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

	trs, err := htmlquery.QueryAll(doc, "//table[@id='EVENTLIST']/tbody[@class='tablecontent']/tr[@id]")
	if err != nil {
		log.Errorf("failed to query lectures document for rows for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	for i, tr := range trs {
		lecture := &model.Lecture{Categories: categories}
		reLecturerId := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

		tds, err := htmlquery.QueryAll(tr, "/td")
		if err != nil || len(tds) != 6 {
			log.Errorf("failed to parse lecture columns for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
			continue
		}

		// LV-Nr
		a, err := htmlquery.Query(tds[1], "/a")
		if err != nil {
			log.Errorf("failed to get lecture id for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
			continue
		}
		lecture.Id = htmlquery.InnerText(a)

		// Titel
		a, err = htmlquery.Query(tds[2], "/a")
		if err != nil {
			log.Errorf("failed to get lecture title for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
			continue
		}
		lecture.Name = htmlquery.InnerText(a)

		// Gguid
		if href := util.GetHTMLAttrValue("href", a.Attr); href != "" {
			matches := reGguid.FindStringSubmatch(href)
			if len(matches) == 2 {
				lecture.Gguid = matches[1]
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
			log.Errorf("failed to get lecture type for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
			continue
		}
		lecture.Type = htmlquery.InnerText(a)

		// Dozenten
		lecturers := make([]*model.Lecturer, 0)
		as, err := htmlquery.QueryAll(tds[3], "/a")
		if err != nil {
			log.Errorf("failed to get lecturers for tguid %s and gguid %s in row %d\n", l.Tguid, l.Gguid, i)
			continue
		}
		for _, a := range as {
			lecturer := &model.Lecturer{}
			lecturer.Name = htmlquery.InnerText(a)

			if href := util.GetHTMLAttrValue("href", a.Attr); href != "" {
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

		lecture.Lecturers = lecturers
		lectures = append(lectures, lecture)
	}

	return lectures, nil
}
