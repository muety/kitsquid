package main

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/model"
	"github.com/n1try/kithub2/scraping"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()
}

func main() {
	scraper := scraping.NewLectureScraper()
	job := scraping.FetchLecturesJob{Semester: model.SemesterWs1819}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	log.Flush()
}
