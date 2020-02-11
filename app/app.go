package app

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/model"
	"github.com/n1try/kithub2/app/scraping"
	"github.com/n1try/kithub2/app/store"
	"github.com/n1try/kithub2/app/web"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	config.Init()
	store.Init()
	web.Init()
}

func readCategories(atIndex, minItems int) []string {
	categoryMap := make(map[string]bool)
	lectures, err := store.GetLectures()
	if err != nil {
		log.Fatalf("failed to read categories")
	}
	for _, l := range lectures {
		if len(l.Categories) >= minItems {
			categoryMap[l.Categories[atIndex]] = true
		}
	}

	result := make([]string, len(categoryMap))
	i := 0
	for k, _ := range categoryMap {
		result[i] = k
		i++
	}
	return result
}

func _debugScrape() {
	scraper := scraping.NewLectureScraper()
	job := scraping.FetchLecturesJob{Semester: model.SemesterWs1819}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	if err := store.InsertLectures(result.([]*model.Lecture), true); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func _debugScrapeDetails() {
	scraper := scraping.NewLectureDetailsScraper()
	ls, err := store.GetLectures()
	if err != nil {
		panic(err)
	}
	job := scraping.FetchDetailsJob{Lectures: ls}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	for _, l := range result.([]*model.Lecture) {
		if err := store.InsertLecture(l, true); err != nil {
			log.Errorf("failed to update lecture %s – %v\n", l.Id, err)
		}
	}
}

func _debugGet() {
	ls, err := store.FindLectures(&model.LectureQuery{
		LecturerIdEq: "0xE4129354DF0A3A4E8EE42CD65C5BCD1C",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println(ls)
}

func Run() {
	web.Start()
	//_debugScrape()
	//_debugScrapeDetails()
	//_debugGet()
}
