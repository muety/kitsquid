package scraping

import (
	"context"
	"errors"
	"github.com/antchfx/htmlquery"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/model"
	"golang.org/x/sync/semaphore"
	"golang.org/x/text/language"
	"net/url"
	"regexp"
	"sync"
)

const (
	baseUrl      = "https://campus.kit.edu/live-stud/campus/all"
	mainUrl      = baseUrl + "/field.asp"
	facultiesUrl = baseUrl + "/fields.asp?group=Vorlesungsverzeichnis"
	maxWorkers   = 6
)

// TODO: Move to external config or so
var tguids = map[model.SemesterKey]string{
	model.SemesterWs1819: "0x4CB7204338AE4F67A58AFCE6C29D1488",
}

type ScrapeJob interface {
	Process() (interface{}, error)
}

type ListFaculties struct {
	Tguid    string
	Language language.Tag
}

type ListCategories struct {
	Tguid string
	Gguid string
}

type ListLectures struct {
	Tguid       string
	Gguid       string
	ParentGguid string
}

type KITCampusPortalScraper struct{}

func (s *KITCampusPortalScraper) FetchLectures(semester model.SemesterKey, lang language.Tag) ([]*model.Lecture, error) {
	var lectures = make([]*model.Lecture, 0)
	var categories = make([]*model.Category, 0)
	var faculties = make([]*model.Faculty, 0)

	makeError := func(err error) ([]*model.Lecture, error) {
		return lectures, err
	}

	if _, ok := tguids[semester]; !ok {
		return makeError(errors.New("unknown semester key"))
	}
	tguid := tguids[semester]

	job1 := ListFaculties{Tguid: tguid, Language: lang}
	result1, err := job1.Process()
	if err != nil {
		return makeError(err)
	}
	faculties = result1.([]*model.Faculty)

	facultyByCategory := make(map[string]string)
	for _, faculty := range faculties {
		job2 := ListCategories{Tguid: tguid, Gguid: faculty.Id}
		result2, err := job2.Process()
		if err != nil {
			log.Errorf("failed to fetch categories – %v\n", err)
			continue
		}
		categories = append(categories, result2.([]*model.Category)...)

		for _, cat := range result2.([]*model.Category) {
			facultyByCategory[cat.Id] = faculty.Id
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
				var found bool
				for _, fid := range item.FacultyIds {
					if fid == l.FacultyIds[0] {
						found = true
						break
					}
				}
				if !found {
					item.FacultyIds = append(item.FacultyIds, l.FacultyIds[0])
				}
			}
		}
	}

	for _, cat := range categories {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Errorf("failed to acquire semaphore while fetching lectures – %v\n", err)
			continue
		}

		catId := cat.Id

		go func() {
			defer sem.Release(1)
			job := ListLectures{Tguid: tguid, Gguid: catId, ParentGguid: facultyByCategory[catId]}
			result, err := job.Process()
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

func (l ListFaculties) Process() (interface{}, error) {
	faculties := make([]*model.Faculty, 0)
	re := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

	u, _ := url.Parse(facultiesUrl)
	q := u.Query()
	q.Add("tguid", l.Tguid)
	q.Add("lang", l.Language.String())
	u.RawQuery = q.Encode()

	log.V(2).Infof("[ListFaculties] processing %s\n", u.String())
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
		var name string
		var href string

		name = htmlquery.InnerText(a)
		for _, attr := range a.Attr {
			if attr.Key == "href" {
				href = attr.Val
				break
			}
		}

		matches := re.FindStringSubmatch(href)
		if len(matches) == 2 {
			faculties = append(faculties, &model.Faculty{
				Name: name,
				Id:   matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return faculties, nil
}

func (l ListCategories) Process() (interface{}, error) {
	categories := make([]*model.Category, 0)
	re := regexp.MustCompile(`.*gguid=(0x[\w\d]+).*`)

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[ListCategories] processing %s\n", u.String())
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
		var name string
		var href string

		name = htmlquery.InnerText(a)
		for _, attr := range a.Attr {
			if attr.Key == "href" {
				href = attr.Val
				break
			}
		}

		matches := re.FindStringSubmatch(href)
		if len(matches) == 2 {
			categories = append(categories, &model.Category{
				Name: name,
				Id:   matches[1], // gguid
			})
		} else {
			log.Errorf("failed to find gguid for %s\n", href)
		}
	}

	return categories, nil
}

func (l ListLectures) Process() (interface{}, error) {
	lectures := make([]*model.Lecture, 0)

	u, _ := url.Parse(mainUrl)
	q := url.Values{}
	q.Add("tguid", l.Tguid)
	q.Add("gguid", l.Gguid)
	q.Add("view", "list")
	q.Add("pagesize", "250")
	u.RawQuery = q.Encode()

	log.V(2).Infof("[ListLectures] processing %s\n", u.String())
	doc, err := htmlquery.LoadURL(u.String())
	if err != nil {
		log.Errorf("failed to load %s\n", u.String())
		return nil, err
	}

	trs, err := htmlquery.QueryAll(doc, "//table[@id='EVENTLIST']/tbody[@class='tablecontent']/tr[@id]")
	if err != nil {
		log.Errorf("failed to query lectures document for tguid %s and gguid %s\n", l.Tguid, l.Gguid)
		return nil, err
	}

	for i, tr := range trs {
		lecture := &model.Lecture{FacultyIds: []string{l.ParentGguid}}
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

			for _, attr := range a.Attr {
				if attr.Key == "href" {
					matches := reLecturerId.FindStringSubmatch(attr.Val)
					if len(matches) == 2 {
						lecturer.Id = matches[1]
						break
					} else {
						log.Errorf("failed to find lecturer gguid for %s\n", attr.Val)
					}
				}
			}

			if lecturer.Id == "" {
				break
			}

			lecturers = append(lecturers, lecturer)
		}

		lecture.Lecturers = lecturers
		lectures = append(lectures, lecture)
	}

	return lectures, nil
}

func NewKITCampusPortalScraper() *KITCampusPortalScraper {
	return &KITCampusPortalScraper{}
}
