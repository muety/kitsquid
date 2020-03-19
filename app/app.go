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
	uuid "github.com/satori/go.uuid"
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
	//_debugScrapeDetails()
	//_debugGet()
	//_debugMail()
}

func _debugScrape() {
	scraper := scraping.NewEventScraper()
	job := scraping.FetchEventsJob{Semester: common.SemesterWs1920}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	if err := events.InsertMulti(result.([]*events.Event), true, false); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func _debugScrapeDetails() {
	scraper := scraping.NewEventDetailsScraper()
	el, err := events.Find(&events.EventQuery{})
	job := scraping.FetchDetailsJob{Events: el}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	for _, l := range result.([]*events.Event) {
		if err := events.Insert(l, true, false); err != nil {
			log.Errorf("failed to update lecture %s – %v\n", l.Id, err)
		}
	}
}

func _debugGet() {
	ls, err := events.Find(&events.EventQuery{
		SemesterEq: string(common.SemesterWs1920),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println(ls)
}

func _debugMail() {
	err := users.SendConfirmationMail(&users.User{
		Id: "ferdinand@muetsch.io",
	}, uuid.NewV4().String())
	if err != nil {
		log.Fatal(err)
	}
}
