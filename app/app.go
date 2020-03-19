package app

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/admin"
	"github.com/n1try/kitsquid/app/comments"
	"github.com/n1try/kitsquid/app/common"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/events"
	"github.com/n1try/kitsquid/app/reviews"
	"github.com/n1try/kitsquid/app/scraping"
	"github.com/n1try/kitsquid/app/users"
	"github.com/n1try/kitsquid/app/web"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	config.Init()
	web.Init()
	events.Init(config.Db())
	users.Init(config.Db())
	comments.Init(config.Db())
	reviews.Init(config.Db())
	admin.Init(config.Db())
}

func Run() {
	web.Start()
	//_debugScrape()
}

func _debugScrape() {
	mainScraper := scraping.NewEventScraper()
	mainJob := scraping.FetchEventsJob{Semester: common.SemesterWs1920}
	mainResult, err := mainScraper.Run(mainJob)
	if err != nil {
		panic(err)
	}
	log.Flush()

	if err := events.InsertMulti(mainResult.([]*events.Event), true, false); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	detailsScraper := scraping.NewEventDetailsScraper()
	detailsJob := scraping.FetchDetailsJob{Events: mainResult.([]*events.Event)}
	detailsResult, err := detailsScraper.Run(detailsJob)
	if err != nil {
		panic(err)
	}
	log.Flush()

	for _, l := range detailsResult.([]*events.Event) {
		if err := events.Insert(l, true, false); err != nil {
			log.Errorf("failed to update lecture %s – %v\n", l.Id, err)
		}
	}
}
