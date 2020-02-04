package app

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/model"
	"github.com/n1try/kithub2/app/scraping"
	"github.com/n1try/kithub2/app/store"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	config.Init()
	store.Init()
}

func Run() {
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

	ls, err := store.GetLectures()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println(ls)
}
