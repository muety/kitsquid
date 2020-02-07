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
}
