package app

import (
	"flag"
	"github.com/n1try/kithub2/app/config"
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

func Run() {
	/*scraper := scraping.NewLectureScraper()
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

	fmt.Println(ls)*/

	web.Start()
}
